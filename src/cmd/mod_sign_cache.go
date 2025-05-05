package cmd

import (
	"github.com/illikainen/gofer/src/mod"
	"github.com/illikainen/gofer/src/sandbox"

	"github.com/illikainen/go-cryptor/src/blob"
	"github.com/illikainen/go-utils/src/flag"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var modSignCacheOpts struct {
	output flag.Path
	goSums flag.PathSlice
}

var modSignCacheCmd = &cobra.Command{
	Use:     "sign-cache [flags] <go.sum>...",
	Short:   "Verify and sign GOPATH modules and their metadata referenced in the specified go.sum file(s)",
	PreRunE: modSignCachePreRun,
	RunE:    modSignCacheRun,
	Args:    cobra.MinimumNArgs(1),
}

func init() {
	flags := modSignCacheCmd.Flags()

	modSignCacheOpts.output.State = flag.MustBeDir | flag.MustNotExist
	modSignCacheOpts.output.Mode = flag.ReadWriteMode
	flags.VarP(&modSignCacheOpts.output, "output", "o", "Output directory for archived modules")
	lo.Must0(modSignCacheCmd.MarkFlagRequired("output"))

	flags.VarP(&modSignCacheOpts.goSums, "go-sums", "", "Go.sum files to parse")
	lo.Must0(flags.MarkHidden("go-sums"))

	modCmd.AddCommand(modSignCacheCmd)
}

func modSignCachePreRun(cmd *cobra.Command, args []string) (err error) {
	for _, arg := range args {
		err := modSignCacheOpts.goSums.Set(arg)
		if err != nil {
			return err
		}
	}

	return sandbox.Exec(&sandbox.SandboxOptions{
		Subcommand: cmd.CalledAs(),
		Flags:      cmd.Flags(),
	})
}

func modSignCacheRun(cmd *cobra.Command, args []string) (err error) {
	cmd.SilenceUsage = true

	keys, err := blob.ReadKeyring(rootOpts.privKey.String(), rootOpts.pubKeys.StringSlice())
	if err != nil {
		return err
	}

	sum, err := mod.ReadGoSum(&mod.SumOptions{
		SumFiles: args,
		SigPath:  modSignCacheOpts.output.String(),
		GoPath:   rootOpts.goPath.String(),
		Log:      log.StandardLogger(),
	})
	if err != nil {
		return err
	}

	err = sum.VerifyAndSign(keys)
	if err != nil {
		return err
	}

	log.Infof("successfully wrote signed cache to %s", modSignCacheOpts.output.String())
	return nil
}
