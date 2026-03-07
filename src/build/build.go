package build

import (
	"path/filepath"
	"runtime"
	"strings"

	"github.com/illikainen/gofer/src/git"
	"github.com/illikainen/gofer/src/gox"

	"github.com/illikainen/go-utils/src/errorx"
	"github.com/illikainen/go-utils/src/iofs"
	log "github.com/sirupsen/logrus"
)

type Options struct {
	Packages []string
	Input    string
	Output   string
	Targets  []string
	Release  bool
}

func Run(opts *Options) (err error) {
	input := opts.Input
	flags := []string{"-mod=readonly", "-trimpath", "-tags", "netgo,osusergo"}
	g := gox.New(&gox.Options{
		Dir:   input,
		Flags: flags,
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
			Flags:   append([]string{"-buildmode=pie", "-ldflags=-s -w -buildid="}, flags...),
			Release: true,
		})
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

	pkgs := strings.Join(opts.Packages, ", ")
	if pkgs == "" {
		pkgs = "."
	}

	for _, target := range opts.Targets {
		parts := strings.Split(target, ":")
		goos := strings.ReplaceAll(parts[0], "host", runtime.GOOS)
		goarch := strings.ReplaceAll(parts[1], "host", runtime.GOARCH)
		dir := filepath.Join(output, goos+"-"+goarch) + string(filepath.Separator)

		log.Infof("building %s to %s", pkgs, dir)
		err = g.Build(opts.Packages, goos, goarch, dir)
		if err != nil {
			return err
		}
	}

	return nil
}
