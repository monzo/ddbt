package cmd

import (
	"fmt"
	"os"

	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(copyCommand)
}

var copyCommand = &cobra.Command{
	Use:   "copy [model name]",
	Short: "Copies the SQL that would be executed for the given model into your clipboard",
	Args:  cobra.ExactValidArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := clipboard.WriteAll(getModelSQL(args[0])); err != nil {
			fmt.Printf("❌ Unable to copy query into your clipboard: %s\n", err)
			os.Exit(1)
		}

		fmt.Printf("📎 Query has been copied into your clipboard\n")
	},
	ValidArgsFunction: completeModelFn,
}
