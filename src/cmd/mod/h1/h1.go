package h1cmd

import (
	"os"
	"path/filepath"
	"strings"

	rootcmd "github.com/illikainen/gofer/src/cmd/root"
	"github.com/illikainen/gofer/src/h1"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var options struct {
	*rootcmd.Options
	module string
}

var command = &cobra.Command{
	Use:     "h1 [flags] <file or directory>",
	Short:   "Compute the H1 for the specified file or directory",
	PreRunE: preRun,
	RunE:    run,
	Args:    cobra.ExactArgs(1),
}

func Command(opts *rootcmd.Options) *cobra.Command {
	options.Options = opts
	return command
}

func init() {
	flags := command.Flags()

	flags.StringVarP(&options.module, "module", "m", "",
		"Specify the module's <name>@v<version>, required when the target is a directory")
}

func preRun(_ *cobra.Command, args []string) error {
	err := options.Sandbox.AddReadOnlyPath(args[0])
	if err != nil {
		return err
	}

	return options.Sandbox.Confine()
}

func run(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	input := args[0]
	stat, err := os.Stat(input)
	if err != nil {
		return err
	}

	if stat.IsDir() {
		if options.module == "" {
			return errors.Errorf("required flag(s) \"module\" not set")
		}

		elts := strings.Split(options.module, "@")
		if len(elts) != 2 {
			return errors.Errorf("invalid <name>@v<version>")
		}

		cksum, err := h1.HashDir(input, elts[0], elts[1])
		if err != nil {
			return err
		}
		log.Infof("%s (dir): %s", input, cksum)
	} else {
		switch filepath.Ext(input) {
		case ".mod":
			cksum, err := h1.HashMod(input)
			if err != nil {
				return err
			}
			log.Infof("%s (mod): %s", input, cksum)
		case ".zip":
			cksum, err := h1.HashZip(input)
			if err != nil {
				return err
			}
			log.Infof("%s (zip): %s", input, cksum)
		default:
			return errors.Errorf("unsupported extension")
		}
	}

	return nil
}
