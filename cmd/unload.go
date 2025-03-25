package cmd

import (
	"github.com/spf13/cobra"
)

var unloadCmd = &cobra.Command{
	Use:   "unload",
	Short: "Unloads the given mods",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

func init() {
	rootCmd.AddCommand(unloadCmd)
}
