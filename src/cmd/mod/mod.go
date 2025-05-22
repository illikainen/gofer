package modcmd

import (
	cachedircmd "github.com/illikainen/gofer/src/cmd/mod/cachedir"
	getcmd "github.com/illikainen/gofer/src/cmd/mod/get"
	h1cmd "github.com/illikainen/gofer/src/cmd/mod/h1"
	signcachecmd "github.com/illikainen/gofer/src/cmd/mod/signcache"
	verifycmd "github.com/illikainen/gofer/src/cmd/mod/verify"
	rootcmd "github.com/illikainen/gofer/src/cmd/root"

	"github.com/spf13/cobra"
)

var command = &cobra.Command{
	Use:   "mod",
	Short: "Module commands",
}

func Command(opts *rootcmd.Options) *cobra.Command {
	command.AddCommand(cachedircmd.Command(opts))
	command.AddCommand(getcmd.Command(opts))
	command.AddCommand(h1cmd.Command(opts))
	command.AddCommand(signcachecmd.Command(opts))
	command.AddCommand(verifycmd.Command(opts))
	return command
}
