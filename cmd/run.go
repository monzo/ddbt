package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

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
	addModelsFlag(runCmd)
}

var runCmd = &cobra.Command{
	Use:     "run",
	Short:   "Runs the DAG",
	Long:    "Run will execute the request DAG",
	Example: "ddbt run -m +my_model",
	Run: func(cmd *cobra.Command, args []string) {
		fileSystem, globalContext := compileAllModels()

		// If we've been given a model to run, run it
		graph := buildGraph(fileSystem, ModelFilter)

		if err := executeGraph(graph, globalContext); err != nil {
			os.Exit(1)
		}
	},
}

func addModelsFlag(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&ModelFilter, "models", "m", "", "Select which model(s) to run")
	err := cmd.RegisterFlagCompletionFunc("models", completeModelFn)
	if err != nil {
		panic(err)
	}
}

func compileAllModels() (*fs.FileSystem, *compiler.GlobalContext) {
	fmt.Printf("ℹ️  Building for %s profile\n", config.GlobalCfg.Target.Name)

	// Read the models on the file system
	fileSystem, err := fs.ReadFileSystem(os.Stdout)
	if err != nil {
		fmt.Printf("❌ Unable to read filesystem: %s\n", err)
		os.Exit(1)
	}

	// Now parse and compile the whole project
	parseFiles(fileSystem)
	gc := compiler.NewGlobalContext(config.GlobalCfg, fileSystem)
	compileMacros(fileSystem, gc)
	compileFiles(fileSystem, gc)
	compileTests(fileSystem, gc)

	return fileSystem, gc
}

func parseFiles(fileSystem *fs.FileSystem) {
	pb := utils.NewProgressBar("📜 Reading & Parsing Files", fileSystem.NumberFiles())
	defer pb.Stop()

	_ = fs.ProcessFiles(
		fileSystem.AllFiles(),
		func(file *fs.File) error {
			if err := compiler.ParseFile(file); err != nil {
				pb.Stop()
				fmt.Printf("❌ Unable to parse %s %s: %s\n", file.Type, file.Name, err)
				os.Exit(1)
			}
			pb.Increment()

			return nil
		},
		nil,
	)
}

func compileMacros(fileSystem *fs.FileSystem, gc *compiler.GlobalContext) {
	pb := utils.NewProgressBar("📚 Compiling Macros", len(fileSystem.Macros()))
	defer pb.Stop()

	_ = fs.ProcessFiles(
		fileSystem.Macros(),
		func(file *fs.File) error {
			err := compiler.CompileModel(file, gc, false)
			if err != nil {
				pb.Stop()
				fmt.Printf("❌ Unable to compile %s %s: %s\n", file.Type, file.Name, err)
				os.Exit(1)
			}
			pb.Increment()

			return nil
		},
		nil,
	)
}

func compileFiles(fileSystem *fs.FileSystem, gc *compiler.GlobalContext) {
	pb := utils.NewProgressBar("📝 Compiling Models", len(fileSystem.Models()))
	defer pb.Stop()

	_ = fs.ProcessFiles(
		fileSystem.Models(),
		func(file *fs.File) error {
			err := compiler.CompileModel(file, gc, false)
			if err != nil {
				pb.Stop()
				fmt.Printf("❌ Unable to compile %s %s: %s\n", file.Type, file.Name, err)
				os.Exit(1)
			}

			pb.Increment()

			return nil
		},
		nil,
	)
}

func compileTests(fileSystem *fs.FileSystem, gc *compiler.GlobalContext) {
	pb := utils.NewProgressBar("🧪 Compiling Tests", len(fileSystem.Tests()))
	defer pb.Stop()

	_ = fs.ProcessFiles(
		fileSystem.Tests(),
		func(file *fs.File) error {
			err := compiler.CompileModel(file, gc, false)
			if err != nil {
				pb.Stop()
				fmt.Printf("❌ Unable to compile %s %s: %s\n", file.Type, file.Name, err)
				os.Exit(1)
			}

			pb.Increment()

			return nil
		},
		nil,
	)
}

func buildGraph(fileSystem *fs.FileSystem, modelFilter string) *fs.Graph {
	pb := utils.NewProgressBar("🚧 Building DAG", 1)
	defer pb.Stop()

	graph := fs.NewGraph()

	if modelFilter != "" {
		if strings.HasPrefix(modelFilter, "tag:") {
			if err := graph.AddFilesWithTag(fileSystem, modelFilter[4:]); err != nil {
				pb.Stop()
				fmt.Printf("❌ Unable to add models filtered by tag: %s", err)
				os.Exit(1)
			}
		} else {
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
		fmt.Printf("❌ Empty DAG generated for model filter: %s\n", modelFilter)
		os.Exit(1)
	}

	return graph
}

func executeGraph(graph *fs.Graph, globalContext *compiler.GlobalContext) error {
	pb := utils.NewProgressBar("🚀 Executing DAG", graph.Len())
	defer pb.Stop()

	ctx, cancel := context.WithCancel(context.Background())

	return graph.Execute(func(file *fs.File) error {
		if file.Type == fs.ModelFile && file.GetMaterialization() != "ephemeral" {
			if file.IsDynamicSQL() || upstreamProfile != "" {
				if err := compiler.CompileModel(file, globalContext, true); err != nil {
					pb.Stop()
					fmt.Printf("❌ %s\n", err)
					cancel()
					return err
				}
			}

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
				return err
			}
		}

		pb.Increment()

		return nil
	}, config.NumberThreads(), pb)
}
