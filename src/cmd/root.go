package cmd

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

var rootOpts struct {
	configp   flag.Path
	config    *config.Config
	profile   string
	privKey   flag.Path
	pubKeys   flag.PathSlice
	goPath    flag.Path
	goCache   flag.Path
	cacheDir  flag.Path
	verbosity logging.LogLevel
}

var rootCmd = &cobra.Command{
	Use:               metadata.Name(),
	Version:           fmt.Sprintf("%s (%s@%s)", metadata.Version(), metadata.Branch(), metadata.Commit()),
	PersistentPreRunE: rootPreRun,
}

func Command() *cobra.Command {
	return rootCmd
}

func init() {
	flags := rootCmd.PersistentFlags()
	flags.SortFlags = false

	levels := []string{}
	for _, level := range log.AllLevels {
		levels = append(levels, level.String())
	}

	rootOpts.configp.State = flag.MustExist
	flags.Var(&rootOpts.configp, "config", "Configuration file")

	flags.StringVarP(&rootOpts.profile, "profile", "p", "", "Profile to use")

	flags.VarP(&rootOpts.verbosity, "verbosity", "V",
		fmt.Sprintf("Verbosity (%s)", strings.Join(levels, ", ")))

	flags.Var(&rootOpts.privKey, "privkey", "Private key file")
	lo.Must0(flags.MarkHidden("privkey"))

	flags.Var(&rootOpts.pubKeys, "pubkeys", "Public key file(s)")
	lo.Must0(flags.MarkHidden("pubkeys"))

	flags.Var(&rootOpts.goPath, "gopath", "GOPATH")
	lo.Must0(flags.MarkHidden("gopath"))

	rootOpts.goCache.State = flag.MustBeDir
	rootOpts.goCache.Mode = flag.ReadWriteMode
	flags.Var(&rootOpts.goCache, "gocache", "GOCACHE")
	lo.Must0(flags.MarkHidden("gocache"))

	flags.Var(&rootOpts.cacheDir, "cache-dir", "Cache directory")
	lo.Must0(flags.MarkHidden("cache-dir"))

	flags.Bool("help", false, "Help for this command")
}

func rootPreRun(cmd *cobra.Command, _ []string) error {
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

	rootOpts.config, err = config.Read(rootOpts.configp.Value)
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

	cfg := rootOpts.config
	pcfg := cfg.Profiles[rootOpts.profile]

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
