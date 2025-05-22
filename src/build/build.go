package build

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/illikainen/gofer/src/git"
	"github.com/illikainen/gofer/src/gox"
	"github.com/illikainen/gofer/src/mod"

	"github.com/illikainen/go-utils/src/errorx"
	"github.com/illikainen/go-utils/src/iofs"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
)

type Options struct {
	Input   string
	Output  string
	Targets []string
	Release bool
}

func Run(opts *Options) (err error) {
	input := opts.Input
	g := gox.New(&gox.Options{
		Dir:   input,
		Flags: []string{"-mod=readonly", "-trimpath"},
	})

	if opts.Release {
		tmp, rmdir, err := iofs.MkdirTemp()
		if err != nil {
			return err
		}
		defer errorx.Defer(rmdir, &err)

		repo := git.NewClient(&git.Options{
			Dir: tmp,
		})

		err = repo.Clone(opts.Input)
		if err != nil {
			return err
		}

		input = tmp
		g = gox.New(&gox.Options{
			Dir:     tmp,
			Flags:   []string{"-mod=readonly", "-trimpath", "-buildmode=pie", "-ldflags=-s -w -buildid="},
			Release: true,
		})
	}

	modfile, err := mod.ParseMod(filepath.Join(input, "go.mod"))
	if err != nil {
		return err
	}

	log.Info("generating ./...")
	err = g.Generate("./...")
	if err != nil {
		return err
	}

	output, err := filepath.Abs(opts.Output)
	if err != nil {
		return err
	}

	builds := []string{}
	for _, target := range opts.Targets {
		parts := strings.Split(target, ":")
		goos := strings.ReplaceAll(parts[0], "host", runtime.GOOS)
		goarch := strings.ReplaceAll(parts[1], "host", runtime.GOARCH)
		basename := fmt.Sprintf("%s-%s-%s", filepath.Base(modfile.Module.Mod.Path), goos, goarch)
		dst := filepath.Join(output, basename)

		if lo.Contains(builds, dst) {
			continue
		}
		builds = append(builds, dst)

		log.Infof("building %s", dst)
		err = g.Build(dst, goos, goarch)
		if err != nil {
			return err
		}
	}

	return nil
}
