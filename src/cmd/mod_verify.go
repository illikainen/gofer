package cmd

import (
	"path/filepath"

	"github.com/illikainen/gofer/src/mod"
	"github.com/illikainen/gofer/src/sandbox"

	"github.com/illikainen/go-cryptor/src/blob"
	"github.com/illikainen/go-utils/src/flag"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var modVerifyOpts struct {
	input  flag.Path
	goSums flag.PathSlice
}

var modVerifyCmd = &cobra.Command{
	Use:     "verify [flags] <go.sum>...",
	Short:   "Verify modules and metadata referenced in the specified go.sum file(s)",
	PreRunE: modVerifyPreRun,
	RunE:    modVerifyRun,
	Args:    cobra.MinimumNArgs(1),
}

func init() {
	flags := modVerifyCmd.Flags()

	modVerifyOpts.input.State = flag.MustExist
	flags.VarP(&modVerifyOpts.input, "input", "i", "Directory with signed modules and metadata")

	flags.VarP(&modVerifyOpts.goSums, "go-sums", "", "Go.sum files to parse")
	lo.Must0(flags.MarkHidden("go-sums"))

	modCmd.AddCommand(modVerifyCmd)
}

func modVerifyPreRun(cmd *cobra.Command, args []string) (err error) {
	for _, arg := range args {
		err := modVerifyOpts.goSums.Set(arg)
		if err != nil {
			return err
		}
	}

	return sandbox.Exec(&sandbox.SandboxOptions{
		Subcommand: cmd.CalledAs(),
		Flags:      cmd.Flags(),
	})
}

func modVerifyRun(cmd *cobra.Command, args []string) (err error) {
	cmd.SilenceUsage = true

	keys, err := blob.ReadKeyring(rootOpts.privKey.String(), rootOpts.pubKeys.StringSlice())
	if err != nil {
		return err
	}

	input := modVerifyOpts.input.String()
	if input == "" {
		input = filepath.Join(rootOpts.config.CacheDir, "mod")
	}

	sum, err := mod.ReadGoSum(&mod.SumOptions{
		SumFiles: args,
		SigPath:  input,
		GoPath:   rootOpts.goPath.String(),
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

	log.Infof("\nsuccessfully verified module(s) and metadata in %s:", rootOpts.goPath.String())
	log.Infof("    %d Go cache zip sources", len(vr.GoZipSources))
	log.Infof("    %d Go cache dir sources", len(vr.GoDirSources))
	log.Infof("    %d Go cache mod files", len(vr.GoModFiles))
	log.Infof("    %d Go cache info files", len(vr.GoInfoFiles))
	return nil
}
