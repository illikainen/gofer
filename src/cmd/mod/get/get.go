package getcmd

import (
	"path/filepath"
	"strings"

	rootcmd "github.com/illikainen/gofer/src/cmd/root"
	"github.com/illikainen/gofer/src/mod"
	"github.com/illikainen/gofer/src/sandbox"

	"github.com/illikainen/go-cryptor/src/blob"
	"github.com/illikainen/go-utils/src/flag"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var command = &cobra.Command{
	Use:     "get [flags] <go.sum>...",
	Short:   "Download modules and metadata referenced in the specified go.sum file(s)",
	PreRunE: preRun,
	RunE:    run,
	Args:    cobra.MinimumNArgs(1),
}

var options struct {
	*rootcmd.Options
	url    flag.URL
	goSums flag.PathSlice
}

func Command(opts *rootcmd.Options) *cobra.Command {
	options.Options = opts
	return command
}

func init() {
	flags := command.Flags()

	flags.Var(&options.url, "url", "repository url")

	flags.VarP(&options.goSums, "go-sums", "", "Go.sum files to parse")
	lo.Must0(flags.MarkHidden("go-sums"))
}

func preRun(cmd *cobra.Command, args []string) error {
	flags := cmd.Flags()

	cfg := options.Config
	pcfg := cfg.Profiles[options.Profile]

	if err := flag.SetFallback(flags, "url", pcfg.URL, cfg.URL); err != nil {
		return err
	}

	if options.url.Value == nil {
		return errors.Errorf("required flag(s) \"url\" not set")
	}

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

	sum, err := mod.ReadGoSum(&mod.SumOptions{
		SumFiles: args,
		SigPath:  filepath.Join(options.Config.CacheDir, "mod"),
		GoPath:   options.GoPath.String(),
		Log:      log.StandardLogger(),
	})
	if err != nil {
		return err
	}

	err = sum.DownloadAndVerify(options.url.Value, keys)
	if err != nil {
		return err
	}

	log.Infof("successfully retrieved module(s) and metadata in %s", strings.Join(args, ", "))
	return nil
}
