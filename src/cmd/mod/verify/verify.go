package verifycmd

import (
	"path/filepath"

	rootcmd "github.com/illikainen/gofer/src/cmd/root"
	"github.com/illikainen/gofer/src/mod"

	"github.com/illikainen/go-cryptor/src/blob"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var options struct {
	*rootcmd.Options
	input string
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

	flags.StringVarP(&options.input, "input", "i", "", "Directory with signed modules and metadata")
}

func preRun(_ *cobra.Command, args []string) error {
	err := options.Sandbox.AddReadOnlyPath(append([]string{options.input}, args...)...)
	if err != nil {
		return err
	}

	return options.Sandbox.Confine()
}

func run(cmd *cobra.Command, args []string) (err error) {
	cmd.SilenceUsage = true

	keys, err := blob.ReadKeyring(options.PrivKey, options.PubKeys)
	if err != nil {
		return err
	}

	input := options.input
	if input == "" {
		input = filepath.Join(options.Config.CacheDir, "mod")
	}

	sum, err := mod.ReadGoSum(&mod.SumOptions{
		SumFiles: args,
		SigPath:  input,
		GoPath:   options.GoPath,
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

	log.Infof("\nsuccessfully verified module(s) and metadata in %s:", options.GoPath)
	log.Infof("    %d Go cache zip sources", len(vr.GoZipSources))
	log.Infof("    %d Go cache dir sources", len(vr.GoDirSources))
	log.Infof("    %d Go cache mod files", len(vr.GoModFiles))
	log.Infof("    %d Go cache info files", len(vr.GoInfoFiles))
	return nil
}
