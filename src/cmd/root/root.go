package rootcmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/illikainen/gofer/src/config"
	"github.com/illikainen/gofer/src/metadata"

	"github.com/illikainen/go-utils/src/flag"
	"github.com/illikainen/go-utils/src/logging"
	"github.com/illikainen/go-utils/src/sandbox"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type Options struct {
	Configp   flag.Path
	Config    *config.Config
	Profile   string
	PrivKey   flag.Path
	PubKeys   flag.PathSlice
	GoPath    flag.Path
	GoCache   flag.Path
	CacheDir  flag.Path
	Verbosity logging.LogLevel
}

var options = Options{}

var command = &cobra.Command{
	Use:               metadata.Name(),
	Version:           metadata.Version(),
	PersistentPreRunE: preRun,
}

func Command() (*cobra.Command, *Options) {
	return command, &options
}

func init() {
	flags := command.PersistentFlags()
	flags.SortFlags = false

	levels := []string{}
	for _, level := range log.AllLevels {
		levels = append(levels, level.String())
	}

	options.Configp.State = flag.MustExist
	flags.Var(&options.Configp, "config", "Configuration file")

	flags.StringVarP(&options.Profile, "profile", "p", "", "Profile to use")

	flags.VarP(&options.Verbosity, "verbosity", "V",
		fmt.Sprintf("Verbosity (%s)", strings.Join(levels, ", ")))

	flags.Var(&options.PrivKey, "privkey", "Private key file")
	lo.Must0(flags.MarkHidden("privkey"))

	flags.Var(&options.PubKeys, "pubkeys", "Public key file(s)")
	lo.Must0(flags.MarkHidden("pubkeys"))

	options.GoPath.State = flag.MustBeDir
	options.GoPath.Mode = flag.ReadWriteMode
	flags.Var(&options.GoPath, "gopath", "GOPATH")
	lo.Must0(flags.MarkHidden("gopath"))

	options.GoCache.State = flag.MustBeDir
	options.GoCache.Mode = flag.ReadWriteMode
	flags.Var(&options.GoCache, "gocache", "GOCACHE")
	lo.Must0(flags.MarkHidden("gocache"))

	options.CacheDir.State = flag.MustBeDir
	options.CacheDir.Mode = flag.ReadWriteMode
	flags.Var(&options.CacheDir, "cache-dir", "Cache directory")
	lo.Must0(flags.MarkHidden("cache-dir"))

	flags.Bool("help", false, "Help for this command")
}

func preRun(cmd *cobra.Command, _ []string) error {
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	cfgPath, err := config.ConfigFile()
	if err != nil {
		return err
	}

	flags := cmd.Flags()
	if err := flag.SetFallback(flags, "config", cfgPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	options.Config, err = config.Read(options.Configp.Value)
	if err != nil {
		return err
	}

	gocmd := exec.Command("go", "env", "GOPATH")
	goPath, err := gocmd.Output()
	if err != nil {
		return err
	}
	if err := flag.SetFallback(flags, "gopath", strings.Trim(string(goPath), "\r\n")); err != nil {
		return err
	}

	gocmd = exec.Command("go", "env", "GOCACHE")
	goCache, err := gocmd.Output()
	if err != nil {
		return err
	}
	if err := flag.SetFallback(flags, "gocache", strings.Trim(string(goCache), "\r\n")); err != nil {
		return err
	}

	cfg := options.Config
	pcfg := cfg.Profiles[options.Profile]

	if err := flag.SetFallback(flags, "verbosity", pcfg.Verbosity, cfg.Verbosity); err != nil {
		return err
	}
	if err := flag.SetFallback(flags, "privkey", pcfg.PrivKey, cfg.PrivKey); err != nil {
		return err
	}
	if err := flag.SetFallback(flags, "pubkeys", pcfg.PubKeys, cfg.PubKeys); err != nil {
		return err
	}
	if err := flag.SetFallback(flags, "cache-dir", pcfg.CacheDir, cfg.CacheDir); err != nil {
		return err
	}

	if !sandbox.IsSandboxed() {
		cmd.SilenceUsage = false
	}
	return nil
}
