package verifycmd

import (
	"path/filepath"

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
	input  flag.Path
	goSums flag.PathSlice
}

var command = &cobra.Command{
	Use:     "verify [flags] <go.sum>...",
	Short:   "Verify modules and metadata referenced in the specified go.sum file(s)",
	PreRunE: preRun,
	RunE:    run,
	Args:    cobra.MinimumNArgs(1),
}

func Command(opts *rootcmd.Options) *cobra.Command {
	options.Options = opts
	return command
}

func init() {
	flags := command.Flags()

	options.input.State = flag.MustExist
	flags.VarP(&options.input, "input", "i", "Directory with signed modules and metadata")

	flags.VarP(&options.goSums, "go-sums", "", "Go.sum files to parse")
	lo.Must0(flags.MarkHidden("go-sums"))
}

func preRun(cmd *cobra.Command, args []string) (err error) {
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

func run(cmd *cobra.Command, args []string) (err error) {
	cmd.SilenceUsage = true

	keys, err := blob.ReadKeyring(options.PrivKey.String(), options.PubKeys.StringSlice())
	if err != nil {
		return err
	}

	input := options.input.String()
	if input == "" {
		input = filepath.Join(options.Config.CacheDir, "mod")
	}

	sum, err := mod.ReadGoSum(&mod.SumOptions{
		SumFiles: args,
		SigPath:  input,
		GoPath:   options.GoPath.String(),
		Log:      log.StandardLogger(),
	})
	if err != nil {
		return err
	}

	vr, err := sum.Verify(keys)
	if err != nil {
		return err
	}

	log.Infof("\nsuccessfully verified module(s) and metadata in %s:", input)
	log.Infof("    %d signed files", len(vr.SignedFiles))
	log.Infof("        %d signed sources", len(vr.SignedSources))
	log.Infof("        %d signed mod files", len(vr.SignedModFiles))
	log.Infof("        %d signed info files", len(vr.SignedInfoFiles))
	if len(vr.SignedFiles) == len(vr.SignedSources)+len(vr.SignedModFiles)+len(vr.SignedInfoFiles) {
		log.Info("        (all signed files also had their content fully verified)")
	}

	log.Infof("\nsuccessfully verified module(s) and metadata in %s:", options.GoPath.String())
	log.Infof("    %d Go cache zip sources", len(vr.GoZipSources))
	log.Infof("    %d Go cache dir sources", len(vr.GoDirSources))
	log.Infof("    %d Go cache mod files", len(vr.GoModFiles))
	log.Infof("    %d Go cache info files", len(vr.GoInfoFiles))
	return nil
}
