package cmd

import (
	buildcmd "github.com/illikainen/gofer/src/cmd/build"
	genkeycmd "github.com/illikainen/gofer/src/cmd/genkey"
	modcmd "github.com/illikainen/gofer/src/cmd/mod"
	rootcmd "github.com/illikainen/gofer/src/cmd/root"
	runcmd "github.com/illikainen/gofer/src/cmd/run"

	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	c, opts := rootcmd.Command()
	c.AddCommand(buildcmd.Command(opts))
	c.AddCommand(genkeycmd.Command(opts))
	c.AddCommand(modcmd.Command(opts))
	c.AddCommand(runcmd.Command(opts))
	return c
}
