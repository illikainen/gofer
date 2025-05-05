package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/illikainen/gofer/src/h1"
	"github.com/illikainen/gofer/src/sandbox"

	"github.com/illikainen/go-utils/src/flag"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var modH1Opts struct {
	module string
	input  flag.Path
}

var modH1Cmd = &cobra.Command{
	Use:     "h1 [flags] <file or directory>",
	Short:   "Compute the H1 for the specified file or directory",
	PreRunE: modH1PreRun,
	RunE:    modH1Run,
	Args:    cobra.ExactArgs(1),
}

func init() {
	flags := modH1Cmd.Flags()

	flags.StringVarP(&modH1Opts.module, "module", "m", "",
		"Specify the module's <name>@v<version>, required when the target is a directory")

	modH1Opts.input.State = flag.MustExist
	flags.VarP(&modH1Opts.input, "input", "", "File or directory to H1")
	lo.Must0(flags.MarkHidden("input"))

	modCmd.AddCommand(modH1Cmd)
}

func modH1PreRun(cmd *cobra.Command, args []string) (err error) {
	for _, arg := range args {
		err := modH1Opts.input.Set(arg)
		if err != nil {
			return err
		}
	}

	return sandbox.Exec(&sandbox.SandboxOptions{
		Subcommand: cmd.CalledAs(),
		Flags:      cmd.Flags(),
	})
}

func modH1Run(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true

	stat, err := os.Stat(modH1Opts.input.String())
	if err != nil {
		return err
	}

	input := modH1Opts.input.String()
	if stat.IsDir() {
		if modH1Opts.module == "" {
			return errors.Errorf("required flag(s) \"module\" not set")
		}

		elts := strings.Split(modH1Opts.module, "@")
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
