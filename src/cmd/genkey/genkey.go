package genkeycmd

import (
	"fmt"
	"time"

	rootcmd "github.com/illikainen/gofer/src/cmd/root"

	"github.com/illikainen/go-cryptor/src/asymmetric"
	"github.com/illikainen/go-utils/src/flag"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var command = &cobra.Command{
	Use:   "genkey",
	Short: "Generate a keypair",
	RunE:  run,
}

var options struct {
	*rootcmd.Options
	output flag.Path
	delay  time.Duration
}

func Command(opts *rootcmd.Options) *cobra.Command {
	options.Options = opts
	return command
}

func init() {
	flags := command.Flags()

	options.output.State = flag.MustNotExist
	options.output.Mode = flag.ReadWriteMode
	options.output.Suffixes = []string{"pub", "priv"}
	flags.VarP(&options.output, "output", "o",
		"Write the generated keypair to <output>.pub and <output>.priv")
	lo.Must0(command.MarkFlagRequired("output"))

	flags.DurationVarP(&options.delay, "delay", "d", 60*time.Second,
		"Add a delay between each generated key")
}

func run(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true

	pubKey, privKey, err := asymmetric.GenerateKey(options.delay)
	if err != nil {
		return err
	}

	pubFile := fmt.Sprintf("%s.pub", options.output.String())
	err = pubKey.Write(pubFile)
	if err != nil {
		return err
	}

	privFile := fmt.Sprintf("%s.priv", options.output.String())
	err = privKey.Write(privFile)
	if err != nil {
		return err
	}

	log.Infof("successfully wrote %s to %s and %s", pubKey.Fingerprint(), pubFile, privFile)
	return nil
}
