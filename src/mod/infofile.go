package mod

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/illikainen/gofer/src/metadata"

	"github.com/illikainen/go-cryptor/src/blob"
	"github.com/illikainen/go-cryptor/src/cryptor"
	"github.com/illikainen/go-utils/src/errorx"
	"github.com/illikainen/go-utils/src/iofs"
	"github.com/illikainen/go-utils/src/logging"
	"github.com/illikainen/go-utils/src/stringx"
	"github.com/pkg/errors"
)

// Unlike .mod files and module code, .info files aren't pinned by hash.
// However, their content follows a simple JSON structure that's strictly
// validated here.
type InfoFile struct {
	Name     string // e.g. github.com/BurntSushi/toml
	Version  string // e.g. v1.3.2
	GoPath   string // e.g. $HOME/go
	sigPath  string // e.g. $HOME/.cache/gofer/mod
	log      logging.Logger
	verified bool
}

func (i *InfoFile) Verify(file string) error {
	data, err := iofs.ReadFile(file)
	if err != nil {
		return err
	}

	if !bytes.Equal(stringx.Sanitize(data), data) {
		return errors.Errorf("invalid content in %s", file)
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	var info Info
	err = decoder.Decode(&info)
	if err != nil {
		return err
	}

	err = info.Verify()
	if err != nil {
		return err
	}

	i.verified = true
	i.log.Tracef("%s: successfully verified json", file)
	return nil
}

func (i *InfoFile) Sign(src string, dst string, keyring *blob.Keyring) (err error) {
	if !i.verified {
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

func (i *InfoFile) DownloadAndVerify(uri *url.URL, sigOutput string, goOutput string,
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

		tmpInfoPath := filepath.Join(tmp, "tmp-info")
		err = iofs.Copy(tmpInfoPath, blobber)
		if err != nil {
			return nil, "", err
		}

		err = i.Verify(tmpInfoPath)
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
	i.log.Tracef("%s: signed by: %s", sigOutput, blobber.Signer)

	tmpInfoPath := filepath.Join(tmp, "info")
	err = iofs.Copy(tmpInfoPath, blobber)
	if err != nil {
		return nil, "", err
	}

	err = i.Verify(tmpInfoPath)
	if err != nil {
		return nil, "", err
	}

	infoPathExists, err := iofs.Exists(goOutput)
	if err != nil {
		return nil, "", err
	}

	if !infoPathExists {
		err := iofs.MoveFile(tmpInfoPath, goOutput)
		if err != nil {
			return nil, "", err
		}
	}

	err = i.Verify(goOutput)
	if err != nil {
		return nil, "", err
	}

	return blobber.Signer, "json", nil
}

// Name of the signed .info file.
func (i *InfoFile) SigName() string {
	return fmt.Sprintf("%s@%s.info.gopkg", strings.ReplaceAll(i.Name, "/", "@"), i.Version)
}

// Name where this utility stores signed files.
func (i *InfoFile) SigPath() string {
	return filepath.Join(i.sigPath, i.SigName())
}

// Name that Go uses for the downloaded .info file.
func (i *InfoFile) InfoName() string {
	return fmt.Sprintf("%s.info", i.Version)
}

// Path where Go caches the downloaded .info file.
func (i *InfoFile) InfoPath() string {
	return filepath.Join(i.GoPath, "pkg", "mod", "cache", "download", downcase(i.Name), "@v", i.InfoName())
}

func (i *InfoFile) String() string {
	return fmt.Sprintf("%s@%s.info", i.Name, i.Version)
}

type Info struct {
	Version string
	Time    string
	Origin  Origin
}

type Origin struct {
	VCS    string
	URL    string
	Ref    string
	Hash   string
	Subdir string
}

func (c *Info) Verify() error {
	err := c.validateVersion()
	if err != nil {
		return err
	}

	err = c.validateTime()
	if err != nil {
		return err
	}

	err = c.validateVCS()
	if err != nil {
		return err
	}

	err = c.validateURL()
	if err != nil {
		return err
	}

	err = c.validateRef()
	if err != nil {
		return err
	}

	err = c.validateHash()
	if err != nil {
		return err
	}

	err = c.validateSubdir()
	if err != nil {
		return err
	}

	return nil
}

func (c *Info) validateVersion() error {
	matched, err := regexp.MatchString(`^v[a-z0-9.-]+$`, c.Version)
	if err != nil {
		return err
	}
	if !matched || strings.Contains(c.Version, "..") {
		return errors.Errorf("invalid version: %s", c.Version)
	}
	return nil
}

func (c *Info) validateTime() error {
	matched, err := regexp.MatchString(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$`, c.Time)
	if err != nil {
		return err
	}
	if !matched {
		return errors.Errorf("invalid time: %s", c.Time)
	}

	return nil
}

func (c *Info) validateVCS() error {
	if c.Origin.VCS != "" && c.Origin.VCS != "git" {
		return errors.Errorf("invalid origin.vcs: %s", c.Origin.VCS)
	}
	return nil
}

func (c *Info) validateURL() error {
	if c.Origin.URL != "" {
		urls := []string{
			`https://cloud\.google\.com/[a-zA-Z0-9/-]+`,
			`https://dario\.cat/[a-zA-Z0-9/-]+`,
			`https://github\.com/[a-zA-Z0-9/-]+`,
			`https://go\.googlesource\.com/[a-zA-Z0-9/-]+`,
			`https://golang\.org/[a-zA-Z0-9/-]+`,
			`https://gopkg\.in/[a-zA-Z0-9/-]+`,
			`https://honnef\.co/[a-zA-Z0-9/-]+`,
			`https://rsc\.io/[a-zA-Z0-9/-]+`,
		}
		matched, err := regexp.MatchString("^("+strings.Join(urls, "|")+")$", c.Origin.URL)
		if err != nil {
			return err
		}
		if !matched {
			return errors.Errorf("invalid origin.url: %s", c.Origin.URL)
		}
	}
	return nil
}

func (c *Info) validateRef() error {
	if c.Origin.Ref != "" {
		matched, err := regexp.MatchString(`^refs/tags/v?[a-z0-9.-]+$`, c.Origin.Ref)
		if err != nil {
			return err
		}
		if !matched || strings.Contains(c.Origin.Ref, "..") {
			return errors.Errorf("invalid origin.ref: %s", c.Origin.Ref)
		}
	}
	return nil
}

func (c *Info) validateHash() error {
	if c.Origin.Hash != "" {
		matched, err := regexp.MatchString(`^[0-9a-f]{40}$`, c.Origin.Hash)
		if err != nil {
			return err
		}
		if !matched {
			return errors.Errorf("invalid origin.hash: %s", c.Origin.Hash)
		}
	}
	return nil
}

func (c *Info) validateSubdir() error {
	if c.Origin.Subdir != "" {
		matched, err := regexp.MatchString(`^[a-z0-9/]+$`, c.Origin.Subdir)
		if err != nil {
			return err
		}
		if !matched {
			return errors.Errorf("invalid origin.subdir: %s", c.Origin.Subdir)
		}
	}
	return nil
}
