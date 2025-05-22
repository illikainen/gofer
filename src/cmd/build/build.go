package buildcmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/illikainen/gofer/src/build"
	rootcmd "github.com/illikainen/gofer/src/cmd/root"
	"github.com/illikainen/gofer/src/mod"
	"github.com/illikainen/gofer/src/sandbox"

	"github.com/illikainen/go-utils/src/flag"
	"github.com/illikainen/go-utils/src/iofs"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

var options struct {
	*rootcmd.Options
	targets []string
	release bool
	output  flag.Path
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

	options.output.Mode = flag.ReadWriteMode
	options.output.State = flag.MustBeDir
	flags.VarP(&options.output, "output", "o", "Output directory")
	lo.Must0(command.MarkFlagRequired("output"))
}

func preRun(cmd *cobra.Command, _ []string) error {
	ro := []string{}
	rw := []string{"."}

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
				ro = append(ro, path)
			}
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	ro = append(ro, cwd)

	cfg, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	ro = append(ro, filepath.Join(cfg, "go"))

	return sandbox.Exec(&sandbox.SandboxOptions{
		Subcommand: cmd.CalledAs(),
		Flags:      cmd.Flags(),
		RO:         ro,
		RW:         rw,
	})
}

func run(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	err = build.Run(&build.Options{
		Input:   cwd,
		Output:  options.output.String(),
		Targets: options.targets,
		Release: options.release,
	})
	if err != nil {
		return err
	}

	return nil
}
