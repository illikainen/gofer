package mod

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/illikainen/gofer/src/h1"
	"github.com/illikainen/gofer/src/metadata"

	"github.com/illikainen/go-cryptor/src/blob"
	"github.com/illikainen/go-cryptor/src/cryptor"
	"github.com/illikainen/go-utils/src/errorx"
	"github.com/illikainen/go-utils/src/iofs"
	"github.com/illikainen/go-utils/src/logging"
	"github.com/pkg/errors"
	"golang.org/x/mod/modfile"
)

type ModFile struct {
	Name      string // e.g. github.com/BurntSushi/toml
	Version   string // e.g. v1.3.2
	Checksum  string // e.g. CxXYINrC8qIiEnFrOxCa7Jy5BFHlXnUU2pbicEuybxQ=
	GoPath    string // e.g. $HOME/go
	sigPath   string // e.g. $HOME/.cache/gofer/mod
	InfoFiles []*InfoFile
	log       logging.Logger
	verified  bool
}

func (m *ModFile) Verify(file string) error {
	err := h1.VerifyMod(file, m.Checksum)
	if err != nil {
		return err
	}

	err = m.parse(file)
	if err != nil {
		return err
	}

	m.verified = true
	m.log.Tracef("%s: successfully verified %s", file, m.Checksum)
	return nil
}

func (m *ModFile) parse(file string) error {
	mod, err := ParseMod(file)
	if err != nil {
		return err
	}

	m.InfoFiles = m.InfoFiles[:]
	m.InfoFiles = append(m.InfoFiles, &InfoFile{
		Name:    m.Name,
		Version: m.Version,
		GoPath:  m.GoPath,
		sigPath: m.sigPath,
		log:     m.log,
	})

	for _, req := range mod.Require {
		name, err := validateName(req.Mod.Path)
		if err != nil {
			return err
		}

		version, _, err := validateVersion(req.Mod.Version)
		if err != nil {
			return err
		}

		m.InfoFiles = append(m.InfoFiles, &InfoFile{
			Name:    name,
			Version: version,
			GoPath:  m.GoPath,
			sigPath: m.sigPath,
			log:     m.log,
		})
	}
	return nil
}

func (m *ModFile) Sign(src string, dst string, keyring *blob.Keyring) (err error) {
	if !m.verified {
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

	input, err := os.Open(src) // #nosec G304
	if err != nil {
		return err
	}
	defer errorx.Defer(input.Close, &err)

	err = iofs.Copy(blobber, input)
	if err != nil {
		return err
	}

	return nil
}

func (m *ModFile) DownloadAndVerify(uri *url.URL, sigOutput string, goOutput string,
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

		tmpModPath := filepath.Join(tmp, "tmp-mod")
		err = iofs.Copy(tmpModPath, blobber)
		if err != nil {
			return nil, "", err
		}

		err = m.Verify(tmpModPath)
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
	m.log.Tracef("%s: signed by: %s", sigOutput, blobber.Signer)

	tmpModPath := filepath.Join(tmp, "mod")
	err = iofs.Copy(tmpModPath, blobber)
	if err != nil {
		return nil, "", err
	}

	err = m.Verify(tmpModPath)
	if err != nil {
		return nil, "", err
	}

	modPathExists, err := iofs.Exists(goOutput)
	if err != nil {
		return nil, "", err
	}

	if !modPathExists {
		err := iofs.MoveFile(tmpModPath, goOutput)
		if err != nil {
			return nil, "", err
		}
	}

	err = m.Verify(goOutput)
	if err != nil {
		return nil, "", err
	}

	return blobber.Signer, m.Checksum, nil
}

// Name of the signed .mod file.
func (m *ModFile) SigName() string {
	return fmt.Sprintf("%s@%s.mod.gopkg", strings.ReplaceAll(m.Name, "/", "@"), m.Version)
}

// Name where this utility stores signed files.
func (m *ModFile) SigPath() string {
	return filepath.Join(m.sigPath, m.SigName())
}

// Name that Go uses for the downloaded .mod file.
func (m *ModFile) ModName() string {
	return fmt.Sprintf("%s.mod", m.Version)
}

// Path where Go caches the downloaded .mod file.
func (m *ModFile) ModPath() string {
	return filepath.Join(m.GoPath, "pkg", "mod", "cache", "download", downcase(m.Name), "@v", m.ModName())
}

// Name for log messages.
func (m *ModFile) String() string {
	return fmt.Sprintf("%s@%s.mod", m.Name, m.Version)
}

func ParseMod(file string) (*modfile.File, error) {
	data, err := iofs.ReadFile(file)
	if err != nil {
		return nil, err
	}

	return modfile.Parse(file, data, nil)
}
