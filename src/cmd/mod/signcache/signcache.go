package signcachecmd

import (
	rootcmd "github.com/illikainen/gofer/src/cmd/root"
	"github.com/illikainen/gofer/src/mod"

	"github.com/illikainen/go-cryptor/src/blob"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var options struct {
	*rootcmd.Options
	output string
}

var command = &cobra.Command{
	Use:     "sign-cache [flags] <go.sum>...",
	Short:   "Verify and sign GOPATH modules and their metadata referenced in the specified go.sum file(s)",
	PreRunE: modSignCachePreRun,
	RunE:    modSignCacheRun,
	Args:    cobra.MinimumNArgs(1),
}

func Command(opts *rootcmd.Options) *cobra.Command {
	options.Options = opts
	return command
}

func init() {
	flags := command.Flags()

	flags.StringVarP(&options.output, "output", "o", "", "Output directory for archived modules")
	lo.Must0(command.MarkFlagRequired("output"))
}

func modSignCachePreRun(_ *cobra.Command, args []string) error {
	err := options.Sandbox.AddReadOnlyPath(args...)
	if err != nil {
		return err
	}

	err = options.Sandbox.AddReadWritePath(options.output)
	if err != nil {
		return err
	}

	return options.Sandbox.Confine()
}

func modSignCacheRun(cmd *cobra.Command, args []string) (err error) {
	cmd.SilenceUsage = true

	keys, err := blob.ReadKeyring(options.PrivKey, options.PubKeys)
	if err != nil {
		return err
	}

	sum, err := mod.ReadGoSum(&mod.SumOptions{
		SumFiles: args,
		SigPath:  options.output,
		GoPath:   options.GoPath,
		Log:      log.StandardLogger(),
	})
	if err != nil {
		return err
	}

	err = sum.VerifyAndSign(keys)
	if err != nil {
		return err
	}

	log.Infof("successfully wrote signed cache to %s", options.output)
	return nil
}
