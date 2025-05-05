package git

import (
	"strconv"
	"strings"

	"github.com/illikainen/go-utils/src/process"
	"github.com/illikainen/go-utils/src/stringx"
	"github.com/pkg/errors"
)

type Options struct {
	Dir string
}

type Git struct {
	*Options
}

func NewClient(opts *Options) *Git {
	return &Git{Options: opts}
}

func (g *Git) Clone(repo string) error {
	_, err := process.Exec(&process.ExecOptions{
		Command: []string{
			"git",
			"clone",
			repo,
			g.Dir,
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func (g *Git) CommitHash(obj string) (string, error) {
	out, err := process.Exec(&process.ExecOptions{
		Command: []string{"git", "-C", g.Dir, "rev-parse", obj},
		Stdout:  process.CaptureOutput,
	})
	if err != nil {
		return "", err
	}

	h := strings.Trim(string(out.Stdout), " \r\n")
	if len(h) != 40 {
		return "", errors.Errorf("invalid hash: %s", h)
	}

	return h, nil
}

func (g *Git) CommitDate(obj string) (int64, error) {
	out, err := process.Exec(&process.ExecOptions{
		Command: []string{"git", "-C", g.Dir, "show", "--no-patch", "--format=%ct", obj},
		Stdout:  process.CaptureOutput,
	})
	if err != nil {
		return 0, err
	}

	lines := stringx.SplitLines(string(out.Stdout))
	date, err := strconv.ParseInt(lines[len(lines)-1], 10, 64)
	if err != nil {
		return 0, err
	}
	if date < 0 {
		return 0, errors.Errorf("invalid date: %d", date)
	}

	return date, nil
}
