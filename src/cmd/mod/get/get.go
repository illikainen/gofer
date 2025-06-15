package getcmd

import (
	"net/url"
	"path/filepath"
	"strings"

	rootcmd "github.com/illikainen/gofer/src/cmd/root"
	"github.com/illikainen/gofer/src/mod"

	"github.com/illikainen/go-cryptor/src/blob"
	"github.com/illikainen/go-utils/src/seq"
	"github.com/pkg/errors"
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
	url string
}

func Command(opts *rootcmd.Options) *cobra.Command {
	options.Options = opts
	return command
}

func init() {
	flags := command.Flags()

	flags.StringVarP(&options.url, "url", "", "", "repository url")
}

func preRun(_ *cobra.Command, args []string) error {
	uri, ok := seq.Coalesce(options.url, options.URL)
	if !ok || uri == "" {
		return errors.Errorf("required flag(s) \"url\" not set")
	}

	u, err := url.Parse(uri)
	if err != nil {
		return err
	}

	if u.Scheme == "file" {
		err := options.Sandbox.AddReadOnlyPath(u.Path)
		if err != nil {
			return err
		}
	}

	err = options.Sandbox.AddReadOnlyPath(args...)
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

	sum, err := mod.ReadGoSum(&mod.SumOptions{
		SumFiles: args,
		SigPath:  filepath.Join(options.Config.CacheDir, "mod"),
		GoPath:   options.GoPath,
		Log:      log.StandardLogger(),
	})
	if err != nil {
		return err
	}

	err = sum.DownloadAndVerify(options.url, keys)
	if err != nil {
		return err
	}

	log.Infof("successfully retrieved module(s) and metadata in %s", strings.Join(args, ", "))
	return nil
}
