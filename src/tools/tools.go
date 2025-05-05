package tools

import (
	"bytes"
	"embed"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/illikainen/gofer/src/mod"

	"github.com/illikainen/go-cryptor/src/blob"
	"github.com/illikainen/go-utils/src/errorx"
	"github.com/illikainen/go-utils/src/iofs"
	log "github.com/sirupsen/logrus"
)

//go:embed tools.mod
//go:embed tools.sum
//go:embed gosec.go
var embeds embed.FS

type tool struct {
	bin      string
	module   string
	pkg      string
	embedded bool
}

var tools = []*tool{
	{bin: "nilerr", module: "github.com/gostaticanalysis/nilerr", pkg: "cmd/nilerr"},
	{bin: "errcheck", module: "github.com/kisielk/errcheck"},
	{bin: "revive", module: "github.com/mgechev/revive"},
	{bin: "goimports", module: "golang.org/x/tools", pkg: "cmd/goimports"},
	{bin: "staticcheck", module: "honnef.co/go/tools", pkg: "cmd/staticcheck"},
	{bin: "gosec", module: "github.com/securego/gosec/v2", pkg: "gosec.go", embedded: true},
}

type ToolOptions struct {
	Bin     string
	BinDir  string
	Args    []string
	SigPath string
	GoPath  string
	Keyring *blob.Keyring
}

func Exec(opts *ToolOptions) error {
	bin := opts.Bin

	for _, tool := range tools {
		if tool.bin == opts.Bin {
			exists, err := iofs.Exists(filepath.Join(opts.BinDir, opts.Bin))
			if err != nil {
				return err
			}
			if !exists {
				err := build(tool, opts)
				if err != nil {
					return err
				}
			}

			bin = filepath.Join(opts.BinDir, opts.Bin)
			break
		}
	}

	cmd := exec.Command(bin, opts.Args...) // #nosec G204
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	log.Debugf("running %s %s", bin, strings.Join(opts.Args, " "))
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func build(tool *tool, opts *ToolOptions) (err error) {
	tmp, rmdir, err := iofs.MkdirTemp()
	if err != nil {
		return err
	}
	defer errorx.Defer(rmdir, &err)

	moddata, err := embeds.ReadFile("tools.mod")
	if err != nil {
		return err
	}

	modfile := filepath.Join(tmp, "go.mod")
	err = iofs.WriteFile(modfile, bytes.NewBuffer(moddata))
	if err != nil {
		return err
	}

	sumdata, err := embeds.ReadFile("tools.sum")
	if err != nil {
		return err
	}

	sumfile := filepath.Join(tmp, "go.sum")
	err = iofs.WriteFile(sumfile, bytes.NewBuffer(sumdata))
	if err != nil {
		return err
	}

	sum, err := mod.ReadGoSum(&mod.SumOptions{
		SumFiles: []string{sumfile},
		SigPath:  opts.SigPath,
		GoPath:   opts.GoPath,
	})
	if err != nil {
		return err
	}

	_, err = sum.Verify(opts.Keyring)
	if err != nil {
		return err
	}

	for _, src := range sum.Sources {
		if tool.module != src.Name {
			continue
		}

		srcpath := filepath.Join(src.DirPath(), tool.pkg)
		if tool.embedded {
			tooldata, err := embeds.ReadFile(tool.pkg)
			if err != nil {
				return err
			}

			srcpath = filepath.Join(tmp, tool.pkg)
			err = iofs.WriteFile(srcpath, bytes.NewBuffer(tooldata))
			if err != nil {
				return err
			}
		}

		dstpath, err := filepath.Abs(filepath.Join(opts.BinDir, opts.Bin))
		if err != nil {
			return err
		}

		cmd := exec.Command("go", "build", "-modfile", modfile, "-o", dstpath, srcpath) // #nosec G204
		cmd.Dir = tmp
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		log.Infof("building '%s@%s' to '%s'", src.Name, src.Version, dstpath)
		err = cmd.Run()
		if err != nil {
			return err
		}
	}

	return nil
}
