package gox

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/illikainen/gofer/src/metadata"

	"github.com/illikainen/go-utils/src/process"
)

type Options struct {
	Dir     string
	Flags   []string
	Release bool
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
			fmt.Sprintf("%s_RELEASE=%v", strings.ToUpper(metadata.Name()), opts.Release),
		),
	}
}

func (g *Go) Generate(target string) error {
	_, err := process.Exec(&process.ExecOptions{
		Command: []string{"go", "generate", target},
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
		Env: append(
			g.env,
			fmt.Sprintf("GOOS=%s", goos),
			fmt.Sprintf("GOARCH=%s", goarch),
		),
		Dir:    g.Dir,
		Stdout: process.LogrusOutput,
		Stderr: process.LogrusOutput,
	})
	return err
}

func GoPath() (string, error) {
	cmd := exec.Command("go", "env", "GOPATH")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.Trim(string(out), "\r\n"), nil
}

func GoCache() (string, error) {
	cmd := exec.Command("go", "env", "GOCACHE")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.Trim(string(out), "\r\n"), nil
}
