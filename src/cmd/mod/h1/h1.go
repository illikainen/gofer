package h1cmd

import (
	"os"
	"path/filepath"
	"strings"

	rootcmd "github.com/illikainen/gofer/src/cmd/root"
	"github.com/illikainen/gofer/src/h1"
	"github.com/illikainen/gofer/src/sandbox"

	"github.com/illikainen/go-utils/src/flag"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var options struct {
	*rootcmd.Options
	module string
	input  flag.Path
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

	options.input.State = flag.MustExist
	flags.VarP(&options.input, "input", "", "File or directory to H1")
	lo.Must0(flags.MarkHidden("input"))
}

func preRun(cmd *cobra.Command, args []string) (err error) {
	for _, arg := range args {
		err := options.input.Set(arg)
		if err != nil {
			return err
		}
	}

	return sandbox.Exec(&sandbox.SandboxOptions{
		Subcommand: cmd.CalledAs(),
		Flags:      cmd.Flags(),
	})
}

func run(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true

	stat, err := os.Stat(options.input.String())
	if err != nil {
		return err
	}

	input := options.input.String()
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
