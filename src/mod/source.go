package mod

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/illikainen/gofer/src/git"
	"github.com/illikainen/gofer/src/h1"
	"github.com/illikainen/gofer/src/metadata"

	"github.com/illikainen/go-cryptor/src/blob"
	"github.com/illikainen/go-cryptor/src/cryptor"
	"github.com/illikainen/go-utils/src/errorx"
	"github.com/illikainen/go-utils/src/iofs"
	"github.com/illikainen/go-utils/src/logging"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/mod/module"
	"golang.org/x/mod/zip"
)

type Source struct {
	Name     string // e.g. github.com/BurntSushi/toml
	Version  string // e.g. v1.3.2
	Checksum string // e.g. o7IhLm0Msx3BaB+n3Ag7L8EVlByGnpq14C4YWiu/gL8=
	GoPath   string // e.g. $HOME/go
	sigPath  string // e.g. $HOME/.cache/gofer/mod
	log      logging.Logger
	verified bool
}

const (
	DirMode = iota
	ZipMode
)

func (s *Source) Verify(path string, mode int) error {
	switch mode {
	case DirMode:
		err := h1.VerifyDir(path, s.Name, s.Version, s.Checksum)
		if err != nil {
			return err
		}
	case ZipMode:
		err := h1.VerifyZip(path, s.Checksum)
		if err != nil {
			return err
		}
	default:
		return errors.Errorf("invalid verification mode")
	}

	zipHashPathExists, err := iofs.Exists(s.ZipHashPath())
	if err != nil {
		return err
	}
	if zipHashPathExists {
		zipHash, err := iofs.ReadFile(s.ZipHashPath())
		if err != nil {
			return err
		}

		if string(zipHash) != s.Checksum {
			return errors.Errorf("invalid content in %s", s.ZipHashPath())
		}
	}

	s.verified = true
	s.log.Tracef("%s: successfully verified: %s", path, s.Checksum)
	return nil
}

func (s *Source) Sign(src string, dst string, keyring *blob.Keyring) (err error) {
	if !s.verified {
		return errors.Errorf("%s has not been verified", src)
	}

	output, err := os.Create(dst) // #nosec G304
	if err != nil {
		return err
	}
	defer errorx.Defer(output.Close, &err)

	blobber, err := blob.NewWriter(output, &blob.Options{
		Type:      metadata.Name(),
		Keyring:   keyring,
		Encrypted: false,
	})
	if err != nil {
		return err
	}
	defer errorx.Defer(blobber.Close, &err)

	return zip.CreateFromDir(blobber, module.Version{
		Path:    s.Name,
		Version: s.Version,
	}, src)
}

func (s *Source) DownloadAndVerify(uri *url.URL, sigOutput string, goOutput string, goHashOutput string,
	keyring *blob.Keyring) (signer cryptor.PublicKey, verified string, err error) {
	tmp, tmpRm, err := iofs.MkdirTemp()
	if err != nil {
		return nil, "", err
	}
	defer errorx.Defer(tmpRm, &err)

	sigPathExists, err := iofs.Exists(sigOutput)
	if err != nil {
		return nil, "", err
	}

	if !sigPathExists {
		tmpSigPath := filepath.Join(tmp, "tmp-sig")
		tmpSig, err := os.OpenFile(tmpSigPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600) // #nosec G304
		if err != nil {
			return nil, "", err
		}
		defer errorx.Defer(tmpSig.Close, &err)

		blobber, err := blob.Download(uri, tmpSig, &blob.Options{
			Type:      metadata.Name(),
			Keyring:   keyring,
			Encrypted: false,
		})
		if err != nil {
			return nil, "", err
		}

		tmpZipPath := filepath.Join(tmp, "tmp-zip")
		err = iofs.Copy(tmpZipPath, blobber)
		if err != nil {
			return nil, "", err
		}

		err = s.Verify(tmpZipPath, ZipMode)
		if err != nil {
			return nil, "", err
		}

		err = iofs.MoveFile(tmpSigPath, sigOutput)
		if err != nil {
			return nil, "", err
		}
	}

	sig, err := os.Open(sigOutput) // #nosec G304
	if err != nil {
		return nil, "", err
	}
	defer errorx.Defer(sig.Close, &err)

	blobber, err := blob.NewReader(sig, &blob.Options{
		Type:      metadata.Name(),
		Keyring:   keyring,
		Encrypted: false,
	})
	if err != nil {
		return nil, "", err
	}
	s.log.Tracef("%s: signed by: %s", sigOutput, blobber.Signer)

	tmpZipPath := filepath.Join(tmp, "zip")
	err = iofs.Copy(tmpZipPath, blobber)
	if err != nil {
		return nil, "", err
	}

	err = s.Verify(tmpZipPath, ZipMode)
	if err != nil {
		return nil, "", err
	}

	zipPathExists, err := iofs.Exists(goOutput)
	if err != nil {
		return nil, "", err
	}

	if !zipPathExists {
		err := iofs.MoveFile(tmpZipPath, goOutput)
		if err != nil {
			return nil, "", err
		}
	}

	zipHashPathExists, err := iofs.Exists(goHashOutput)
	if err != nil {
		return nil, "", err
	}
	if !zipHashPathExists {
		err := iofs.WriteFile(goHashOutput, bytes.NewReader([]byte(s.Checksum)))
		if err != nil {
			return nil, "", err
		}
	}

	err = s.Verify(goOutput, ZipMode)
	if err != nil {
		return nil, "", err
	}

	return blobber.Signer, s.Checksum, nil
}

// Name of the signed codebase.
func (s *Source) SigName() string {
	return fmt.Sprintf("%s@%s.zip.gopkg", strings.ReplaceAll(s.Name, "/", "@"), s.Version)
}

// Name where this utility stores signed files.
func (s *Source) SigPath() string {
	return filepath.Join(s.sigPath, s.SigName())
}

// Name that Go uses for the downloaded .zip file with module code.
func (s *Source) ZipName() string {
	return fmt.Sprintf("%s.zip", s.Version)
}

// Path where Go caches the downloaded .zip file.
func (s *Source) ZipPath() string {
	return filepath.Join(s.GoPath, "pkg", "mod", "cache", "download", downcase(s.Name), "@v", s.ZipName())
}

// Name that Go uses for the .ziphash file that contains the same h1 as
// recorded in go.sum.
func (s *Source) ZipHashName() string {
	return fmt.Sprintf("%s.ziphash", s.Version)
}

// Path where Go caches the .ziphash file.
func (s *Source) ZipHashPath() string {
	return filepath.Join(s.GoPath, "pkg", "mod", "cache", "download", downcase(s.Name), "@v", s.ZipHashName())
}

// Name that Go uses for extracted module code.
func (s *Source) DirName() string {
	return fmt.Sprintf("%s@%s", downcase(s.Name), s.Version)
}

// Path where Go extracts the downloaded .zip file.
func (s *Source) DirPath() string {
	return filepath.Join(s.GoPath, "pkg", "mod", s.DirName())
}

func (s *Source) String() string {
	return fmt.Sprintf("%s@%s", s.Name, s.Version)
}

type CacheResult struct {
	DirH1 string
	ModH1 string
	Path  string
	Mod   module.Version
}

func CacheDir(inDir string, outDir string) (result *CacheResult, err error) {
	modpath := filepath.Join(inDir, "go.mod")
	modfile, err := ParseMod(modpath)
	if err != nil {
		return nil, err
	}
	moduleName := modfile.Module.Mod.Path

	g := git.NewClient(&git.Options{
		Dir: inDir,
	})

	commit, err := g.CommitHash("HEAD")
	if err != nil {
		return nil, err
	}

	epoch, err := g.CommitDate(commit)
	if err != nil {
		return nil, err
	}

	rx, err := regexp.Compile("[ :-]")
	if err != nil {
		return nil, err
	}
	date := rx.ReplaceAllString(time.Unix(epoch, 0).UTC().Format("2006-01-02 15:04:05"), "")
	moduleVersion := fmt.Sprintf("v0.0.0-%s-%s", date, commit[:12])

	tmp, rmdir, err := iofs.MkdirTemp()
	if err != nil {
		return nil, err
	}
	defer errorx.Defer(rmdir, &err)

	repo := filepath.Join(tmp, "repo")
	err = git.NewClient(&git.Options{Dir: repo}).Clone(g.Dir)
	if err != nil {
		return nil, err
	}

	archive := filepath.Join(tmp, "archive.zip")
	archivef, err := os.Create(archive) // #nosec G304
	if err != nil {
		return nil, err
	}
	defer errorx.Defer(archivef.Close, &err)

	err = zip.CreateFromDir(archivef, module.Version{
		Path:    moduleName,
		Version: moduleVersion,
	}, repo)
	if err != nil {
		return nil, err
	}

	err = archivef.Sync()
	if err != nil {
		return nil, err
	}

	dirH1, err := h1.HashZip(archive)
	if err != nil {
		return nil, err
	}

	modH1, err := h1.HashMod(filepath.Join(repo, "go.mod"))
	if err != nil {
		return nil, err
	}

	basedir := filepath.Join(outDir, "pkg", "mod", "cache", "download", downcase(moduleName), "@v")
	dst := filepath.Join(basedir, moduleVersion+".zip")
	log.Tracef("moving %s to %s", archive, dst)

	err = iofs.MoveFile(archive, dst)
	if err != nil {
		return nil, err
	}

	err = iofs.WriteFile(filepath.Join(basedir, moduleVersion+".ziphash"), bytes.NewReader([]byte(dirH1)))
	if err != nil {
		return nil, err
	}

	err = iofs.Copy(filepath.Join(basedir, moduleVersion+".mod"), modpath)
	if err != nil {
		return nil, err
	}

	return &CacheResult{
		DirH1: dirH1,
		ModH1: modH1,
		Path:  basedir,
		Mod: module.Version{
			Path:    moduleName,
			Version: moduleVersion,
		},
	}, nil
}
