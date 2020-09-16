package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"ddbt/bigquery"
)

func init() {
	rootCmd.AddCommand(showCmd)
}

var showCmd = &cobra.Command{
	Use:   "show [model name]",
	Short: "Shows the SQL that would be executed for the given model",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fileSystem := compileAllModels()

		model := fileSystem.Model(args[0])
		if model == nil {
			fmt.Printf("❌ Model %s not found", args[0])
			os.Exit(1)
		}

		fmt.Println(bigquery.BuildQuery(model))
	},
}
