package gox

import (
	"os"

	"github.com/illikainen/go-utils/src/process"
)

type Options struct {
	Dir   string
	Flags []string
}

type Go struct {
	*Options
	env []string
}

func New(opts *Options) *Go {
	return &Go{
		Options: opts,
		env: append(
			os.Environ(),
			"GOFLAGS=-mod=readonly",
			"GOPROXY=off",
		),
	}
}

func (g *Go) Generate(targets ...string) error {
	_, err := process.Exec(&process.ExecOptions{
		Command: append([]string{"go", "generate"}, targets...),
		Env:     g.env,
		Dir:     g.Dir,
		Stdout:  process.LogrusOutput,
		Stderr:  process.LogrusOutput,
	})
	return err
}

func (g *Go) Build(output string, goos string, goarch string) error {
	_, err := process.Exec(&process.ExecOptions{
		Command: append([]string{"go", "build", "-o", output}, g.Flags...),
		Env:     append(g.env, "GOOS="+goos, "GOARCH="+goarch),
		Dir:     g.Dir,
		Stdout:  process.LogrusOutput,
		Stderr:  process.LogrusOutput,
	})
	return err
}
