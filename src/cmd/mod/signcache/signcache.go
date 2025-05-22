package signcachecmd

import (
	rootcmd "github.com/illikainen/gofer/src/cmd/root"
	"github.com/illikainen/gofer/src/mod"
	"github.com/illikainen/gofer/src/sandbox"

	"github.com/illikainen/go-cryptor/src/blob"
	"github.com/illikainen/go-utils/src/flag"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var options struct {
	*rootcmd.Options
	output flag.Path
	goSums flag.PathSlice
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

	options.output.State = flag.MustBeDir | flag.MustNotExist
	options.output.Mode = flag.ReadWriteMode
	flags.VarP(&options.output, "output", "o", "Output directory for archived modules")
	lo.Must0(command.MarkFlagRequired("output"))

	flags.VarP(&options.goSums, "go-sums", "", "Go.sum files to parse")
	lo.Must0(flags.MarkHidden("go-sums"))
}

func modSignCachePreRun(cmd *cobra.Command, args []string) (err error) {
	for _, arg := range args {
		err := options.goSums.Set(arg)
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

	keys, err := blob.ReadKeyring(options.PrivKey.String(), options.PubKeys.StringSlice())
	if err != nil {
		return err
	}

	sum, err := mod.ReadGoSum(&mod.SumOptions{
		SumFiles: args,
		SigPath:  options.output.String(),
		GoPath:   options.GoPath.String(),
		Log:      log.StandardLogger(),
	})
	if err != nil {
		return err
	}

	err = sum.VerifyAndSign(keys)
	if err != nil {
		return err
	}

	log.Infof("successfully wrote signed cache to %s", options.output.String())
	return nil
}
