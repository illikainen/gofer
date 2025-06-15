package rootcmd

import (
	"fmt"
	"strings"

	"github.com/illikainen/gofer/src/config"
	"github.com/illikainen/gofer/src/metadata"

	"github.com/illikainen/go-utils/src/fn"
	"github.com/illikainen/go-utils/src/process"
	"github.com/illikainen/go-utils/src/sandbox"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type Options struct {
	config.Config
	config  string
	Sandbox sandbox.Sandbox
	sandbox string
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

	flags.StringVarP(&options.config, "config", "", fn.Must1(config.ConfigFile()), "Configuration file")
	flags.StringVarP(&options.Profile, "profile", "p", "", "Profile to use")
	flags.StringVarP(&options.PrivKey, "privkey", "", "", "Private key file")
	flags.StringSliceVarP(&options.PubKeys, "pubkeys", "", nil, "Public key file(s)")
	flags.StringVarP(&options.Verbosity, "verbosity", "V", "",
		fmt.Sprintf("Verbosity (%s)", strings.Join(levels, ", ")))
	flags.StringVarP(&options.sandbox, "sandbox", "", "", "Sandbox backend")

	flags.Bool("help", false, "Help for this command")
}

func preRun(cmd *cobra.Command, _ []string) error {
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	cfg, err := config.Read(options.config, &options.Config)
	if err != nil {
		return err
	}
	options.Config = *cfg

	level, err := log.ParseLevel(options.Verbosity)
	if err != nil {
		return err
	}
	log.SetLevel(level)

	if !sandbox.IsSandboxed() {
		cmd.SilenceUsage = false
	}

	name := fn.Ternary(options.sandbox != "", options.sandbox, options.Config.Sandbox)
	backend, err := sandbox.Backend(name)
	if err != nil {
		return err
	}

	switch backend {
	case sandbox.BubblewrapSandbox:
		options.Sandbox, err = sandbox.NewBubblewrap(&sandbox.BubblewrapOptions{
			ReadOnlyPaths: append([]string{
				options.config,
				options.PrivKey,
			}, options.PubKeys...),
			ReadWritePaths: []string{
				options.GoPath,
				options.GoCache,
				options.CacheDir,
			},
			Tmpfs:            true,
			Devtmpfs:         true,
			Procfs:           true,
			AllowCommonPaths: true,
			Stdout:           process.LogrusOutput,
			Stderr:           process.LogrusOutput,
		})
		if err != nil {
			return err
		}
	case sandbox.NoSandbox:
		options.Sandbox, err = sandbox.NewNoop()
		if err != nil {
			return err
		}
	}

	return nil
}
