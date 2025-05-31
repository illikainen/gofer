package mod

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/illikainen/gofer/src/metadata"

	"github.com/illikainen/go-cryptor/src/blob"
	"github.com/illikainen/go-netutils/src/transport"
	"github.com/illikainen/go-utils/src/errorx"
	"github.com/illikainen/go-utils/src/iofs"
	"github.com/illikainen/go-utils/src/logging"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"
)

type SumOptions struct {
	SumFiles []string
	SigPath  string
	GoPath   string
	Log      logging.Logger
}

type SumFile struct {
	Sources  []*Source
	ModFiles []*ModFile
	sigPath  string
	goPath   string
	log      logging.Logger
}

func ReadGoSum(opts *SumOptions) (g *SumFile, err error) {
	gosum := &SumFile{
		sigPath: opts.SigPath,
		goPath:  opts.GoPath,
		log:     lo.Ternary(opts.Log != nil, opts.Log, logging.DiscardLogger()),
	}
	seen := []string{}

	for _, sumfile := range opts.SumFiles {
		data, err := iofs.ReadFile(sumfile)
		if err != nil {
			return nil, err
		}

		scan := bufio.NewScanner(bytes.NewReader(data))
		for scan.Scan() {
			err := scan.Err()
			if err != nil {
				return nil, err
			}

			elts := strings.Split(scan.Text(), " ")
			if len(elts) != 3 {
				return nil, errors.Errorf("invalid line: %s", scan.Text())
			}

			name, err := validateName(elts[0])
			if err != nil {
				return nil, err
			}

			version, mod, err := validateVersion(elts[1])
			if err != nil {
				return nil, err
			}

			cksum, err := validateChecksum(elts[2])
			if err != nil {
				return nil, err
			}

			seenElt := fmt.Sprintf("%s@%s@%s", name, version, cksum)
			if !lo.Contains(seen, seenElt) {
				if mod {
					gosum.ModFiles = append(gosum.ModFiles, &ModFile{
						Name:     name,
						Version:  version,
						Checksum: cksum,
						GoPath:   opts.GoPath,
						sigPath:  opts.SigPath,
						log:      gosum.log,
					})
				} else {
					gosum.Sources = append(gosum.Sources, &Source{
						Name:     name,
						Version:  version,
						Checksum: cksum,
						GoPath:   opts.GoPath,
						sigPath:  opts.SigPath,
						log:      gosum.log,
					})
				}
				seen = append(seen, seenElt)
			}
		}

		err = scan.Err()
		if err != nil {
			return nil, err
		}
	}

	return gosum, nil
}

type VerifyResult struct {
	SignedFiles     []string
	SignedSources   []string
	SignedModFiles  []string
	SignedInfoFiles []string
	GoZipSources    []string
	GoDirSources    []string
	GoModFiles      []string
	GoInfoFiles     []string
}

func (s *SumFile) Verify(keyring *blob.Keyring) (vr *VerifyResult, err error) {
	s.log.Infof("Signature directory: %s", s.sigPath)
	s.log.Infof("GOPATH: %s", s.goPath)
	s.log.Info()

	tmp, tmpRm, err := iofs.MkdirTemp()
	if err != nil {
		return nil, err
	}
	defer errorx.Defer(tmpRm, &err)

	sigFiles, err := os.ReadDir(s.sigPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		sigFiles = []os.DirEntry{}
	}

	// The .info files must be verified after the .mod files have been
	// verified and parsed.
	sort.Slice(sigFiles, func(i int, j int) bool {
		iname := sigFiles[i].Name()
		jname := sigFiles[j].Name()

		if strings.HasSuffix(iname, ".mod.gopkg") && !strings.HasSuffix(jname, ".mod.gopkg") {
			return true
		}

		if !strings.HasSuffix(iname, ".mod.gopkg") && strings.HasSuffix(jname, ".mod.gopkg") {
			return false
		}

		return iname < jname
	})

	align := 0
	aligner := lo.MaxBy(sigFiles, func(a fs.DirEntry, b fs.DirEntry) bool {
		return len(a.Name()) > len(b.Name())
	})
	if aligner != nil {
		align = len(aligner.Name())
	}

	// Verify signed files.
	vr = &VerifyResult{}
	for _, elt := range sigFiles {
		if lo.Contains(vr.SignedFiles, elt.Name()) {
			continue
		}

		// Verify the signatures for all files even if they're not
		// referenced in the go.sum file.
		f, err := os.Open(filepath.Join(s.sigPath, elt.Name()))
		if err != nil {
			return nil, err
		}
		defer errorx.Defer(f.Close, &err) // revive:disable-line:defer

		blobber, err := blob.NewReader(f, &blob.Options{
			Type:      metadata.Name(),
			Keyring:   keyring,
			Encrypted: false,
		})
		if err != nil {
			return nil, err
		}
		s.log.Infof("%-*s: signed by %s", align, elt.Name(), blobber.Signer)

		// If the file is referenced in the go.sum, also verify the
		// content of the signed data.
		for _, src := range s.Sources {
			if src.SigName() == elt.Name() && !lo.Contains(vr.SignedSources, elt.Name()) {
				tmpfile := filepath.Join(tmp, src.ZipName())
				err := iofs.Copy(tmpfile, blobber)
				if err != nil {
					return nil, err
				}

				err = src.Verify(tmpfile, ZipMode)
				if err != nil {
					return nil, err
				}

				s.log.Infof("%-*s: verified %s", align, elt.Name(), src.Checksum)
				vr.SignedSources = append(vr.SignedSources, elt.Name())
			}
		}

		for _, m := range s.ModFiles {
			if m.SigName() == elt.Name() && !lo.Contains(vr.SignedModFiles, elt.Name()) {
				tmpfile := filepath.Join(tmp, m.ModName())
				err := iofs.Copy(tmpfile, blobber)
				if err != nil {
					return nil, err
				}

				err = m.Verify(tmpfile)
				if err != nil {
					return nil, err
				}

				s.log.Infof("%-*s: verified %s", align, elt.Name(), m.Checksum)
				vr.SignedModFiles = append(vr.SignedModFiles, elt.Name())
			}

			for _, i := range m.InfoFiles {
				if i.SigName() == elt.Name() && !lo.Contains(vr.SignedInfoFiles, elt.Name()) {
					tmpfile := filepath.Join(tmp, i.InfoName())
					err := iofs.Copy(tmpfile, blobber)
					if err != nil {
						return nil, err
					}

					err = i.Verify(tmpfile)
					if err != nil {
						return nil, err
					}

					s.log.Infof("%-*s: verified json", align, elt.Name())
					vr.SignedInfoFiles = append(vr.SignedInfoFiles, elt.Name())
				}
			}
		}
		vr.SignedFiles = append(vr.SignedFiles, elt.Name())
	}

	// Verify files in the Go cache.
	for _, src := range s.Sources {
		exists, err := iofs.Exists(src.ZipPath())
		if err != nil {
			return nil, err
		}
		if exists && !lo.Contains(vr.GoZipSources, src.ZipPath()) {
			err := src.Verify(src.ZipPath(), ZipMode)
			if err != nil {
				return nil, err
			}

			s.log.Infof("%-*s: verified %s", align, src.String()+".zip", src.Checksum)
			vr.GoZipSources = append(vr.GoZipSources, src.ZipPath())
		}

		exists, err = iofs.Exists(src.DirPath())
		if err != nil {
			return nil, err
		}
		if exists && !lo.Contains(vr.GoDirSources, src.DirPath()) {
			err := src.Verify(src.DirPath(), DirMode)
			if err != nil {
				return nil, err
			}

			s.log.Infof("%-*s: verified %s", align, src, src.Checksum)
			vr.GoDirSources = append(vr.GoDirSources, src.DirPath())
		}
	}

	for _, m := range s.ModFiles {
		exists, err := iofs.Exists(m.ModPath())
		if err != nil {
			return nil, err
		}
		if exists && !lo.Contains(vr.GoModFiles, m.ModPath()) {
			err := m.Verify(m.ModPath())
			if err != nil {
				return nil, err
			}

			s.log.Infof("%-*s: verified %s", align, m, m.Checksum)
			vr.GoModFiles = append(vr.GoModFiles, m.ModPath())
		}

		for _, i := range m.InfoFiles {
			exists, err := iofs.Exists(i.InfoPath())
			if err != nil {
				return nil, err
			}
			if exists && !lo.Contains(vr.GoInfoFiles, i.InfoPath()) {
				err := i.Verify(i.InfoPath())
				if err != nil {
					return nil, err
				}

				s.log.Infof("%-*s: verified json", align, i)
				vr.GoInfoFiles = append(vr.GoInfoFiles, i.InfoPath())
			}
		}
	}

	if len(sigFiles) != len(vr.SignedFiles) {
		return nil, errors.Errorf("bug")
	}
	return vr, nil
}

func (s *SumFile) VerifyAndSign(keyring *blob.Keyring) error {
	err := os.MkdirAll(s.sigPath, 0700)
	if err != nil {
		return err
	}

	align := len(lo.MaxBy(s.ModFiles, func(a *ModFile, b *ModFile) bool {
		return len(a.String()) > len(b.String())
	}).String())

	for _, src := range s.Sources {
		err := src.Verify(src.DirPath(), DirMode)
		if err != nil {
			return err
		}
		s.log.Infof("%-*s: verified %s", align, src, src.Checksum)

		err = src.Sign(src.DirPath(), src.SigPath(), keyring)
		if err != nil {
			return err
		}
	}

	seen := []string{}
	for _, m := range s.ModFiles {
		err := m.Verify(m.ModPath())
		if err != nil {
			return err
		}
		s.log.Infof("%-*s: verified %s", align, m, m.Checksum)

		for _, i := range m.InfoFiles {
			if lo.Contains(seen, i.String()) {
				continue
			}
			seen = append(seen, i.String())

			exists, err := iofs.Exists(i.InfoPath())
			if err != nil {
				return err
			}

			if !exists {
				continue
			}

			err = i.Verify(i.InfoPath())
			if err != nil {
				return err
			}
			s.log.Infof("%-*s: verified json", align, i)

			err = i.Sign(i.InfoPath(), i.SigPath(), keyring)
			if err != nil {
				return err
			}
		}

		err = m.Sign(m.ModPath(), m.SigPath(), keyring)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *SumFile) DownloadAndVerify(uri string, keyring *blob.Keyring) error {
	group := errgroup.Group{}
	semaphore := make(chan int, 3)

	align := len(lo.MaxBy(s.ModFiles, func(a *ModFile, b *ModFile) bool {
		return len(a.String()) > len(b.String())
	}).String())

	baseuri, err := url.Parse(uri)
	if err != nil {
		return err
	}

	for _, m := range s.ModFiles {
		u, err := baseuri.Parse(filepath.Join(baseuri.Path, m.SigName()))
		if err != nil {
			return err
		}
		m := m

		group.Go(func() error {
			semaphore <- 1
			s.log.Infof("%-*s: download from %s (%d)", align, m, u, align)
			signer, verified, err := m.DownloadAndVerify(u, m.SigPath(), m.ModPath(), keyring)
			if err == nil {
				s.log.Infof("%-*s: signed by %s", align, m, signer)
				s.log.Infof("%-*s: verified %s", align, m, verified)
			}
			<-semaphore
			return err
		})
	}

	err = group.Wait()
	if err != nil {
		return err
	}

	seen := []string{}
	for _, m := range s.ModFiles {
		for _, i := range m.InfoFiles {
			if lo.Contains(seen, i.String()) {
				continue
			}
			seen = append(seen, i.String())

			u, err := baseuri.Parse(filepath.Join(baseuri.Path, i.SigName()))
			if err != nil {
				return err
			}
			i := i

			group.Go(func() error {
				semaphore <- 1
				s.log.Infof("%-*s: download from %s", align, i, u)
				signer, verified, err := i.DownloadAndVerify(u, i.SigPath(), i.InfoPath(), keyring)
				if err == nil {
					s.log.Infof("%-*s: signed by %s", align, i, signer)
					s.log.Infof("%-*s: verified %s", align, i, verified)
				} else if errors.Is(err, transport.ErrNotExist) {
					s.log.Debugf("%-*s: not available", align, i)
					err = nil
				}
				<-semaphore
				return err
			})
		}
	}

	for _, src := range s.Sources {
		u, err := baseuri.Parse(filepath.Join(baseuri.Path, src.SigName()))
		if err != nil {
			return err
		}
		src := src

		group.Go(func() error {
			semaphore <- 1
			s.log.Infof("%-*s: download from %s", align, src, u)
			signer, verified, err := src.DownloadAndVerify(u, src.SigPath(), src.ZipPath(),
				src.ZipHashPath(), keyring)
			if err == nil {
				s.log.Infof("%-*s: signed by %s", align, src, signer)
				s.log.Infof("%-*s: verified %s", align, src, verified)
			}
			<-semaphore
			return err
		})
	}

	return group.Wait()
}
