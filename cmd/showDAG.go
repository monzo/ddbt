package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"ddbt/fs"
	"ddbt/utils"
)

func init() {
	rootCmd.AddCommand(showDAG)
	addModelsFlag(showDAG)
}

var showDAG = &cobra.Command{
	Use:   "show-dag",
	Short: "Shows the order in which the DAG would execute",
	Run: func(cmd *cobra.Command, args []string) {
		fileSystem, _ := compileAllModels()

		// If we've been given a model to run, run it
		graph := buildGraph(fileSystem, ModelFilters)

		printGraph(graph)
	},
}

func printGraph(graph *fs.Graph) {
	pb := utils.NewProgressBar("🔖 Writing DAG out", graph.Len())
	defer pb.Stop()

	var builder strings.Builder

	builder.WriteRune('\n')

	_ = graph.Execute(
		func(file *fs.File) error {
			if file.Type == fs.ModelFile {
				builder.WriteString("- ")
				builder.WriteString(file.Name)
				builder.WriteRune('\n')
			}

			pb.Increment()

			return nil
		},
		1,
		pb,
	)

	pb.Stop()

	fmt.Println(builder.String())
}
