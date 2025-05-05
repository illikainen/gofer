package cmd

import (
	"path/filepath"
	"strings"

	"github.com/illikainen/gofer/src/mod"
	"github.com/illikainen/gofer/src/sandbox"

	"github.com/illikainen/go-cryptor/src/blob"
	"github.com/illikainen/go-utils/src/flag"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var modGetOpts struct {
	url    flag.URL
	goSums flag.PathSlice
}

var modGetCmd = &cobra.Command{
	Use:     "get [flags] <go.sum>...",
	Short:   "Download modules and metadata referenced in the specified go.sum file(s)",
	PreRunE: modGetPreRun,
	RunE:    modGetRun,
	Args:    cobra.MinimumNArgs(1),
}

func init() {
	flags := modGetCmd.Flags()

	flags.Var(&modGetOpts.url, "url", "repository url")

	flags.VarP(&modSignCacheOpts.goSums, "go-sums", "", "Go.sum files to parse")
	lo.Must0(flags.MarkHidden("go-sums"))

	modCmd.AddCommand(modGetCmd)
}

func modGetPreRun(cmd *cobra.Command, args []string) error {
	flags := cmd.Flags()

	cfg := rootOpts.config
	pcfg := cfg.Profiles[rootOpts.profile]

	if err := flag.SetFallback(flags, "url", pcfg.URL, cfg.URL); err != nil {
		return err
	}

	if modGetOpts.url.Value == nil {
		return errors.Errorf("required flag(s) \"url\" not set")
	}

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

func modGetRun(cmd *cobra.Command, args []string) (err error) {
	cmd.SilenceUsage = true

	keys, err := blob.ReadKeyring(rootOpts.privKey.String(), rootOpts.pubKeys.StringSlice())
	if err != nil {
		return err
	}

	sum, err := mod.ReadGoSum(&mod.SumOptions{
		SumFiles: args,
		SigPath:  filepath.Join(rootOpts.config.CacheDir, "mod"),
		GoPath:   rootOpts.goPath.String(),
		Log:      log.StandardLogger(),
	})
	if err != nil {
		return err
	}

	err = sum.DownloadAndVerify(modGetOpts.url.Value, keys)
	if err != nil {
		return err
	}

	log.Infof("successfully retrieved module(s) and metadata in %s", strings.Join(args, ", "))
	return nil
}
