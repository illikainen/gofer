package cachedircmd

import (
	rootcmd "github.com/illikainen/gofer/src/cmd/root"
	"github.com/illikainen/gofer/src/mod"
	"github.com/illikainen/gofer/src/sandbox"

	"github.com/illikainen/go-utils/src/flag"
	"github.com/samber/lo"
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
	input  flag.Path
	output flag.Path
}

func Command(opts *rootcmd.Options) *cobra.Command {
	options.Options = opts
	return command
}

func init() {
	flags := command.Flags()

	options.input.State = flag.MustBeDir
	flags.VarP(&options.input, "input", "", "Directory to cache")
	lo.Must0(flags.MarkHidden("input"))

	options.output.Mode = flag.ReadWriteMode
	options.output.State = flag.MustBeDir
	flags.VarP(&options.output, "output", "o", "Output directory")
	lo.Must0(flags.MarkHidden("output"))
}

func preRun(cmd *cobra.Command, args []string) error {
	err := options.input.Set(args[0])
	if err != nil {
		return err
	}

	if err := flag.SetFallback(cmd.Flags(), "output", options.GoPath.String()); err != nil {
		return err
	}

	return sandbox.Exec(&sandbox.SandboxOptions{
		Subcommand: cmd.CalledAs(),
		Flags:      cmd.Flags(),
	})
}

func run(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true

	rv, err := mod.CacheDir(options.input.String(), options.output.String())
	if err != nil {
		return err
	}

	log.Info("go.sum lines:\n")
	log.Info()
	log.Infof("%s %s %s", rv.Mod.Path, rv.Mod.Version, rv.DirH1)
	log.Infof("%s %s/go.mod %s\n", rv.Mod.Path, rv.Mod.Version, rv.ModH1)
	log.Info()
	log.Infof("successfully cached %s to %s", options.input.String(), rv.Path)
	return nil
}
