package cmd

import (
	"fmt"
	"os"

	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"

	"ddbt/bigquery"
	"ddbt/compiler"
)

func init() {
	rootCmd.AddCommand(copyCommand)
}

var copyCommand = &cobra.Command{
	Use:   "copy [model name]",
	Short: "Copies the SQL that would be executed for the given model into your clipboard",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fileSystem, gc := compileAllModels()

		model := fileSystem.Model(args[0])
		if model == nil {
			fmt.Printf("❌ Model %s not found\n", args[0])
			os.Exit(1)
		}

		if model.IsDynamicSQL() {
			if err := compiler.CompileModel(model, gc, true); err != nil {
				fmt.Printf("❌ Unable to compile dynamic SQL: %s\n", err)
				os.Exit(1)
			}
		}

		if err := clipboard.WriteAll(bigquery.BuildQuery(model)); err != nil {
			fmt.Printf("❌ Unable to copy query into your clipboard: %s\n", err)
			os.Exit(1)
		}

		fmt.Printf("📎 Query has been copied into your clipboard\n")
	},
}
