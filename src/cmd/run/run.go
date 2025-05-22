package runcmd

import (
	"os"
	"path/filepath"
	"strings"

	rootcmd "github.com/illikainen/gofer/src/cmd/root"
	"github.com/illikainen/gofer/src/mod"
	"github.com/illikainen/gofer/src/sandbox"
	"github.com/illikainen/gofer/src/tools"

	"github.com/illikainen/go-cryptor/src/blob"
	"github.com/illikainen/go-utils/src/flag"
	"github.com/illikainen/go-utils/src/iofs"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

var options struct {
	*rootcmd.Options
	bindir flag.Path
}

var command = &cobra.Command{
	Use:   "run [flags] <bin> [--] [args...]",
	Short: "Launch a command in a sandbox on compatible systems",
	Long: "Launch a command in a sandbox on compatible systems.\n\n" +
		"On GNU/Linux systems, this command requires Bubblewrap to execute the " +
		"specified program in a sandboxed environment.  " +
		"If the program is listed in the Gofer go.sum file and has not been built yet, " +
		"it will be compiled before execution.",
	Args:    cobra.MinimumNArgs(1),
	PreRunE: preRun,
	RunE:    run,
}

func Command(opts *rootcmd.Options) *cobra.Command {
	options.Options = opts
	return command
}

func init() {
	flags := command.Flags()

	options.bindir.Mode = flag.ReadWriteMode
	options.bindir.State = flag.MustBeDir
	flags.VarP(&options.bindir, "bindir", "b", "Binary directory for new builds")
	lo.Must0(flags.MarkHidden("bindir"))
}

func preRun(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	err = flag.SetFallback(cmd.Flags(), "bindir", filepath.Join(home, ".local", "bin"))
	if err != nil {
		return err
	}

	ro := []string{}
	rw := []string{"."}

	exists, err := iofs.Exists("go.work")
	if err != nil {
		return err
	}
	if exists {
		work, err := mod.ParseWork("go.work")
		if err != nil {
			return err
		}

		for _, replace := range work.Replace {
			path := replace.New.String()
			if strings.HasPrefix(path, ".") {
				ro = append(ro, path)
			}
		}
	}

	return sandbox.Exec(&sandbox.SandboxOptions{
		Subcommand: cmd.CalledAs(),
		Flags:      cmd.Flags(),
		RO:         ro,
		RW:         rw,
	})
}

func run(_ *cobra.Command, args []string) (err error) {
	keys, err := blob.ReadKeyring(options.PrivKey.String(), options.PubKeys.StringSlice())
	if err != nil {
		return err
	}

	return tools.Exec(&tools.ToolOptions{
		Bin:     args[0],
		BinDir:  options.bindir.String(),
		Args:    args[1:],
		SigPath: filepath.Join(options.Config.CacheDir, "mod"),
		GoPath:  options.GoPath.String(),
		Keyring: keys,
	})
}
