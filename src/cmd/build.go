package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/illikainen/gofer/src/build"
	"github.com/illikainen/gofer/src/mod"
	"github.com/illikainen/gofer/src/sandbox"

	"github.com/illikainen/go-utils/src/flag"
	"github.com/illikainen/go-utils/src/iofs"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

var buildOpts struct {
	targets []string
	release bool
	output  flag.Path
}

var buildCmd = &cobra.Command{
	Use:     "build",
	Short:   "Build a project",
	PreRunE: buildPreRun,
	RunE:    buildRun,
}

func init() {
	flags := buildCmd.Flags()

	flags.StringSliceVarP(&buildOpts.targets, "targets", "t", []string{"host:host"}, "Build targets")

	flags.BoolVarP(&buildOpts.release, "release", "", false, "Do a release build")

	buildOpts.output.Mode = flag.ReadWriteMode
	buildOpts.output.State = flag.MustBeDir
	flags.VarP(&buildOpts.output, "output", "o", "Output directory")
	lo.Must0(buildCmd.MarkFlagRequired("output"))

	rootCmd.AddCommand(buildCmd)
}

func buildPreRun(cmd *cobra.Command, _ []string) error {
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

func buildRun(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	err = build.Run(&build.Options{
		Input:   cwd,
		Output:  buildOpts.output.String(),
		Targets: buildOpts.targets,
		Release: buildOpts.release,
	})
	if err != nil {
		return err
	}

	return nil
}
