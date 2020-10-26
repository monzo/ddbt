package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"ddbt/bigquery"
	"ddbt/compiler"
)

func init() {
	rootCmd.AddCommand(showCmd)
}

var showCmd = &cobra.Command{
	Use:   "show [model name]",
	Short: "Shows the SQL that would be executed for the given model",
	Args:  cobra.ExactValidArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(getModelSQL(args[0]))
	},
	ValidArgsFunction: completeModelFn,
}

func getModelSQL(modelName string) string {
	fileSystem, gc := compileAllModels()

	model := fileSystem.Model(modelName)
	if model == nil {
		fmt.Printf("❌ Model %s not found\n", modelName)
		os.Exit(1)
	}

	if model.IsDynamicSQL() || upstreamProfile != "" {
		if err := compiler.CompileModel(model, gc, true); err != nil {
			fmt.Printf("❌ Unable to compile dynamic SQL: %s\n", err)
			os.Exit(1)
		}
	}

	return bigquery.BuildQuery(model)
}
