package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"

	"ddbt/bigquery"
	"ddbt/compiler"
	"ddbt/config"
	"ddbt/fs"
	"ddbt/utils"
)

var ModelFilter string

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringVarP(&ModelFilter, "model", "m", "", "Select which model to run")
}

var runCmd = &cobra.Command{
	Use:     "run",
	Short:   "Runs the DAG",
	Long:    "Run will execute the request DAG",
	Example: "ddbt run -m +my_model",
	Run: func(cmd *cobra.Command, args []string) {
		fileSystem := compileAllModels()

		// If we've been given a model to run, run it
		graph := buildGraph(fileSystem, ModelFilter)

		executeGraph(graph)
	},
}

func compileAllModels() *fs.FileSystem {
	fmt.Printf("ℹ️ Building for %s (%s.%s)\n", config.GlobalCfg.Target.Name, config.GlobalCfg.Target.ProjectID, config.GlobalCfg.Target.DataSet)

	// Read the models on the file system
	fileSystem, err := fs.ReadFileSystem()
	if err != nil {
		fmt.Printf("❌ Unable to read filesystem: %s\n", err)
		os.Exit(1)
	}

	// Now parse and compile the whole project
	parseFiles(fileSystem)
	gc := compiler.NewGlobalContext(config.GlobalCfg, fileSystem)
	compileMacros(fileSystem, gc)
	compileFiles(fileSystem, gc)

	return fileSystem
}

func parseFiles(fileSystem *fs.FileSystem) {
	pb := utils.NewProgressBar("📜 Reading & Parsing Files", fileSystem.NumberFiles())
	defer pb.Stop()

	fs.ProcessFiles(
		fileSystem.AllFiles(),
		func(file *fs.File) {
			if err := compiler.ParseFile(file); err != nil {
				pb.Stop()
				fmt.Printf("❌ Unable to parse %s %s: %s\n", file.Type, file.Name, err)
				os.Exit(1)
			}
			pb.Increment()
		},
	)
}

func compileMacros(fileSystem *fs.FileSystem, gc *compiler.GlobalContext) {
	pb := utils.NewProgressBar("📚 Compiling Macros", len(fileSystem.Macros()))
	defer pb.Stop()

	fs.ProcessFiles(
		fileSystem.Macros(),
		func(file *fs.File) {
			err := compiler.CompileModel(file, gc)
			if err != nil {
				pb.Stop()
				fmt.Printf("❌ Unable to compile %s %s: %s\n", file.Type, file.Name, err)
				os.Exit(1)
			}
			pb.Increment()
		},
	)
}

func compileFiles(fileSystem *fs.FileSystem, gc *compiler.GlobalContext) {
	pb := utils.NewProgressBar("📝 Compiling Models", len(fileSystem.Models()))
	defer pb.Stop()

	fs.ProcessFiles(
		fileSystem.Models(),
		func(file *fs.File) {
			err := compiler.CompileModel(file, gc)
			if err != nil {
				pb.Stop()
				fmt.Printf("❌ Unable to compile %s %s: %s\n", file.Type, file.Name, err)
				os.Exit(1)
			}

			pb.Increment()
		},
	)
}

func buildGraph(fileSystem *fs.FileSystem, modelFilter string) *fs.Graph {
	pb := utils.NewProgressBar("🕸 Building DAG", 1)
	defer pb.Stop()

	graph := fs.NewGraph()

	if modelFilter != "" {
		// Check if we want all upstreams
		allUpstreams := modelFilter[0] == '+'
		if allUpstreams {
			modelFilter = modelFilter[1:]
		}

		allDownstreams := modelFilter[len(modelFilter)-1] == '+'
		if allDownstreams {
			modelFilter = modelFilter[:len(modelFilter)-1]
		}

		model := fileSystem.Model(modelFilter)
		if model == nil {
			pb.Stop()
			fmt.Printf("❌ Unable to find model: %s\n", modelFilter)
			os.Exit(1)
		}

		graph.AddNode(model)

		if allUpstreams {
			if err := graph.AddNodeAndUpstreams(model); err != nil {
				pb.Stop()
				fmt.Printf("❌ %s\n", err)
				os.Exit(1)
			}
		}

		if allDownstreams {
			if err := graph.AddNodeAndDownstreams(model); err != nil {
				pb.Stop()
				fmt.Printf("❌ %s\n", err)
				os.Exit(1)
			}
		}
	} else {
		if err := graph.AddAllModels(fileSystem); err != nil {
			pb.Stop()
			fmt.Printf("❌ %s\n", modelFilter)
			os.Exit(1)
		}
	}

	pb.Increment()

	if graph.Len() == 0 {
		fmt.Printf("❌ Empty DAG generated for model: %s\n", modelFilter)
		os.Exit(1)
	}

	return graph
}

func executeGraph(graph *fs.Graph) {
	pb := utils.NewProgressBar("🚀 Executing DAG", graph.Len())
	defer pb.Stop()

	ctx, cancel := context.WithCancel(context.Background())

	graph.Execute(func(file *fs.File) {
		if file.Type == fs.ModelFile {
			if queryStr, err := bigquery.Run(ctx, file); err != nil {
				pb.Stop()

				if err != context.Canceled {
					fmt.Printf("❌ %s\n", err)

					if err := clipboard.WriteAll(queryStr); err != nil {
						fmt.Printf("   Unable to copy query to clipboard: %s\n", err)
					} else {
						fmt.Printf("📎 Query has been copied into your clipboard\n\n")
					}
				}

				cancel()
				os.Exit(1)
			}
		}

		pb.Increment()
	}, config.NumberThreads(), pb)
}
