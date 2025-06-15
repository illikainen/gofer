package buildcmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/illikainen/gofer/src/build"
	rootcmd "github.com/illikainen/gofer/src/cmd/root"
	"github.com/illikainen/gofer/src/mod"

	"github.com/illikainen/go-utils/src/fn"
	"github.com/illikainen/go-utils/src/iofs"
	"github.com/spf13/cobra"
)

var options struct {
	*rootcmd.Options
	targets []string
	release bool
	output  string
}

var command = &cobra.Command{
	Use:     "build",
	Short:   "Build a project",
	PreRunE: preRun,
	RunE:    run,
}

func Command(opts *rootcmd.Options) *cobra.Command {
	options.Options = opts
	return command
}

func init() {
	flags := command.Flags()

	flags.StringSliceVarP(&options.targets, "targets", "t", []string{"host:host"}, "Build targets")

	flags.BoolVarP(&options.release, "release", "", false, "Do a release build")

	flags.StringVarP(&options.output, "output", "o", "", "Output directory")
	fn.Must(command.MarkFlagRequired("output"))
}

func preRun(_ *cobra.Command, _ []string) error {
	err := options.Sandbox.AddReadWritePath(".", options.output)
	if err != nil {
		return err
	}

	exists, err := iofs.Exists("go.work")
	if err != nil {
		return err
	}
	if exists {
		work, err := mod.ParseWork("go.work")
		if err != nil {
			return err
		}

		for _, replace := range work.Replace {
			path := replace.New.String()
			if strings.HasPrefix(path, ".") {
				err := options.Sandbox.AddReadOnlyPath(path)
				if err != nil {
					return err
				}
			}
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	cfg, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	err = options.Sandbox.AddReadOnlyPath(cwd, filepath.Join(cfg, "go"))
	if err != nil {
		return err
	}

	return options.Sandbox.Confine()
}

func run(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	err = build.Run(&build.Options{
		Input:   cwd,
		Output:  options.output,
		Targets: options.targets,
		Release: options.release,
	})
	if err != nil {
		return err
	}

	return nil
}
