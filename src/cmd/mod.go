package cmd

import (
	"github.com/spf13/cobra"
)

var modCmd = &cobra.Command{
	Use:   "mod",
	Short: "Module commands",
}

func init() {
	rootCmd.AddCommand(modCmd)
}
