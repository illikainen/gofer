package cmd

import (
	"github.com/illikainen/gofer/src/mod"
	"github.com/illikainen/gofer/src/sandbox"

	"github.com/illikainen/go-utils/src/flag"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var modCacheDirOpts struct {
	input  flag.Path
	output flag.Path
}

var modCacheDirCmd = &cobra.Command{
	Use:     "cache-dir [flags] <dir>",
	Short:   "Create a locally cached version from the *trusted* <dir>",
	PreRunE: modCacheDirPreRun,
	RunE:    modCacheDirRun,
	Args:    cobra.ExactArgs(1),
}

func init() {
	flags := modCacheDirCmd.Flags()

	modCacheDirOpts.input.State = flag.MustBeDir
	flags.VarP(&modCacheDirOpts.input, "input", "", "Directory to cache")
	lo.Must0(flags.MarkHidden("input"))

	modCacheDirOpts.output.Mode = flag.ReadWriteMode
	modCacheDirOpts.output.State = flag.MustBeDir
	flags.VarP(&modCacheDirOpts.output, "output", "o", "Output directory")
	lo.Must0(flags.MarkHidden("output"))

	modCmd.AddCommand(modCacheDirCmd)
}

func modCacheDirPreRun(cmd *cobra.Command, args []string) error {
	err := modCacheDirOpts.input.Set(args[0])
	if err != nil {
		return err
	}

	if err := flag.SetFallback(cmd.Flags(), "output", rootOpts.goPath.String()); err != nil {
		return err
	}

	return sandbox.Exec(&sandbox.SandboxOptions{
		Subcommand: cmd.CalledAs(),
		Flags:      cmd.Flags(),
	})
}

func modCacheDirRun(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true

	rv, err := mod.CacheDir(modCacheDirOpts.input.String(), modCacheDirOpts.output.String())
	if err != nil {
		return err
	}

	log.Info("go.sum lines:\n")
	log.Info()
	log.Infof("%s %s %s", rv.Mod.Path, rv.Mod.Version, rv.DirH1)
	log.Infof("%s %s/go.mod %s\n", rv.Mod.Path, rv.Mod.Version, rv.ModH1)
	log.Info()
	log.Infof("successfully cached %s to %s", modCacheDirOpts.input.String(), rv.Path)
	return nil
}
