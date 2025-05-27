package sandbox

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/illikainen/go-netutils/src/sshx"
	"github.com/illikainen/go-utils/src/errorx"
	"github.com/illikainen/go-utils/src/flag"
	"github.com/illikainen/go-utils/src/iofs"
	"github.com/illikainen/go-utils/src/process"
	"github.com/illikainen/go-utils/src/sandbox"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

type SandboxOptions struct {
	Subcommand string
	Flags      *pflag.FlagSet
	RO         []string
	RW         []string
}

func Exec(opts *SandboxOptions) error {
	if !sandbox.Compatible() || sandbox.IsSandboxed() {
		return nil
	}

	paths := []*flag.Path{}
	opts.Flags.VisitAll(func(f *pflag.Flag) {
		switch v := f.Value.(type) {
		case *flag.Path:
			paths = append(paths, v)
		case *flag.PathSlice:
			paths = append(paths, v.Value...)
		case *flag.URL:
			if v.Value != nil && v.Value.Scheme == "file" {
				paths = append(paths, &flag.Path{
					Value: v.Value.Path,
				})
			}
		}
	})

	ro := append([]string{}, opts.RO...)
	rw := append([]string{}, opts.RW...)
	created := []string{}
	for _, path := range paths {
		if path.String() == "" {
			continue
		}

		if path.Mode == flag.ReadWriteMode {
			newPaths, err := ensurePath(path)
			if err != nil {
				return err
			}
			created = append(created, newPaths...)

			if len(path.Values) <= 0 {
				rw = append(rw, path.Value)
			} else {
				rw = append(rw, path.Values...)
			}
		} else {
			if len(path.Values) <= 0 {
				ro = append(ro, path.Value)
			} else {
				ro = append(ro, path.Values...)
			}
		}
	}

	share := 0
	if opts.Subcommand == "get" {
		share |= sandbox.ShareNet

		sshRO, sshRW, err := sshx.SandboxPaths()
		if err != nil {
			return err
		}
		ro = append(ro, sshRO...)
		rw = append(rw, sshRW...)
	}

	bin, err := os.Executable()
	if err != nil {
		return err
	}

	bin, err = filepath.Abs(bin)
	if err != nil {
		return err
	}

	_, err = sandbox.Exec(sandbox.Options{
		Command: append([]string{bin}, os.Args[1:]...),
		RO:      ro,
		RW:      rw,
		Proc:    true,
		Share:   share,
		Stdout:  process.LogrusOutput,
		Stderr:  process.LogrusOutput,
	})
	if err != nil {
		errs := []error{err}
		for _, path := range created {
			log.Debugf("removing %s", path)
			errs = append(errs, iofs.Remove(path))
		}
		return errorx.Join(errs...)
	}

	os.Exit(0) // revive:disable-line
	return nil
}

func ensurePath(path *flag.Path) ([]string, error) {
	paths := path.Values
	if len(paths) <= 0 {
		paths = append(paths, path.Value)
	}

	created := []string{}
	for _, p := range paths {
		if p == "" {
			continue
		}

		exists, err := iofs.Exists(p)
		if err != nil {
			return created, err
		}
		if exists {
			return created, nil
		}

		if path.State&flag.MustBeDir == flag.MustBeDir {
			dir := p
			parts := strings.Split(p, string(os.PathSeparator))

			for i := len(parts); i > 0; i-- {
				cur := strings.Join(parts[:i], string(os.PathSeparator))
				exists, err := iofs.Exists(cur)
				if err != nil {
					return created, err
				}
				if exists {
					break
				}
				dir = cur
			}

			log.Debugf("creating %s as a directory", p)
			err := os.MkdirAll(p, 0700)
			if err != nil {
				return created, err
			}

			created = append(created, dir)
		} else {
			log.Debugf("creating %s as a regular file", p)
			f, err := os.Create(p) // #nosec G304
			if err != nil {
				return created, err
			}

			created = append(created, p)

			err = f.Close()
			if err != nil {
				return created, err
			}
		}
	}

	return created, nil
}
