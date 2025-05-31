package cachedircmd

import (
	rootcmd "github.com/illikainen/gofer/src/cmd/root"
	"github.com/illikainen/gofer/src/mod"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var command = &cobra.Command{
	Use:     "cache-dir [flags] <dir>",
	Short:   "Create a locally cached version from the *trusted* <dir>",
	PreRunE: preRun,
	RunE:    run,
	Args:    cobra.ExactArgs(1),
}

var options struct {
	*rootcmd.Options
}

func Command(opts *rootcmd.Options) *cobra.Command {
	options.Options = opts
	return command
}

func preRun(_ *cobra.Command, args []string) error {
	err := options.Sandbox.AddReadOnlyPath(args[0])
	if err != nil {
		return err
	}
	return options.Sandbox.Confine()
}

func run(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	rv, err := mod.CacheDir(args[0], options.GoPath)
	if err != nil {
		return err
	}

	log.Info("go.sum lines:\n")
	log.Info()
	log.Infof("%s %s %s", rv.Mod.Path, rv.Mod.Version, rv.DirH1)
	log.Infof("%s %s/go.mod %s\n", rv.Mod.Path, rv.Mod.Version, rv.ModH1)
	log.Info()
	log.Infof("successfully cached %s to %s", args[0], rv.Path)
	return nil
}
