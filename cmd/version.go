package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"ddbt/utils"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints the version of DDBT",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("ddbt version", utils.DdbtVersion)
	},
}
