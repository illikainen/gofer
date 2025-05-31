package genkeycmd

import (
	"fmt"
	"time"

	rootcmd "github.com/illikainen/gofer/src/cmd/root"

	"github.com/illikainen/go-cryptor/src/asymmetric"
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
	output string
	delay  time.Duration
}

func Command(opts *rootcmd.Options) *cobra.Command {
	options.Options = opts
	return command
}

func init() {
	flags := command.Flags()

	flags.StringVarP(&options.output, "output", "o", "",
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

	pubFile := fmt.Sprintf("%s.pub", options.output)
	err = pubKey.Write(pubFile)
	if err != nil {
		return err
	}

	privFile := fmt.Sprintf("%s.priv", options.output)
	err = privKey.Write(privFile)
	if err != nil {
		return err
	}

	log.Infof("successfully wrote %s to %s and %s", pubKey.Fingerprint(), pubFile, privFile)
	return nil
}
