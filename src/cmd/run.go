package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/illikainen/gofer/src/mod"
	"github.com/illikainen/gofer/src/sandbox"
	"github.com/illikainen/gofer/src/tools"

	"github.com/illikainen/go-cryptor/src/blob"
	"github.com/illikainen/go-utils/src/flag"
	"github.com/illikainen/go-utils/src/iofs"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

var runOpts struct {
	bindir flag.Path
}

var runCmd = &cobra.Command{
	Use:   "run [flags] <bin> [--] [args...]",
	Short: "Launch a command in a sandbox on compatible systems",
	Long: "Launch a command in a sandbox on compatible systems.\n\n" +
		"On GNU/Linux systems, this command requires Bubblewrap to execute the " +
		"specified program in a sandboxed environment.  " +
		"If the program is listed in the Gofer go.sum file and has not been built yet, " +
		"it will be compiled before execution.",
	Args:    cobra.MinimumNArgs(1),
	PreRunE: runPreRun,
	RunE:    runRun,
}

func init() {
	flags := runCmd.Flags()

	runOpts.bindir.Mode = flag.ReadWriteMode
	runOpts.bindir.State = flag.MustBeDir
	flags.VarP(&runOpts.bindir, "bindir", "b", "Binary directory for new builds")
	lo.Must0(flags.MarkHidden("bindir"))

	rootCmd.AddCommand(runCmd)
}

func runPreRun(cmd *cobra.Command, _ []string) error {
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

func runRun(_ *cobra.Command, args []string) (err error) {
	keys, err := blob.ReadKeyring(rootOpts.privKey.String(), rootOpts.pubKeys.StringSlice())
	if err != nil {
		return err
	}

	return tools.Exec(&tools.ToolOptions{
		Bin:     args[0],
		BinDir:  runOpts.bindir.String(),
		Args:    args[1:],
		SigPath: filepath.Join(rootOpts.config.CacheDir, "mod"),
		GoPath:  rootOpts.goPath.String(),
		Keyring: keys,
	})
}
