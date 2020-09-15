package main

import (
	"fmt"
	"log"
	"os"

	"ddbt/bigquery"
	"ddbt/compiler"
	"ddbt/config"
	"ddbt/fs"
	"ddbt/utils"
)

func main() {
	fileSystem, err := fs.ReadFileSystem()
	if err != nil {
		log.Fatalf("Unable to read filesystem: %s", err)
	}

	cfg := config.Read()

	parseFiles(fileSystem)
	gc := compiler.NewGlobalContext(cfg, fileSystem)
	compileMacros(fileSystem, gc)
	compileFiles(fileSystem, gc)

	if len(os.Args) > 1 {
		modelName := os.Args[1]

		graph := buildGraph(fileSystem, modelName)

		executeGraph(graph)
	}
}

func parseFiles(fileSystem *fs.FileSystem) {
	pb := utils.NewProgressBar("📜 Reading & Parsing Files", fileSystem.NumberFiles())
	defer pb.Stop()

	utils.ProcessFiles(
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

	utils.ProcessFiles(
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

	utils.ProcessFiles(
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

func buildGraph(fileSystem *fs.FileSystem, modelName string) *fs.Graph {
	pb := utils.NewProgressBar("🕸 Building DAG", 1)
	defer pb.Stop()

	model := fileSystem.Model(modelName)
	if model == nil {
		pb.Stop()
		fmt.Printf("❌ Unable to find model: %s\n", modelName)
		os.Exit(1)
	}

	graph := fs.NewGraph()

	if err := graph.AddTargetNode(model); err != nil {
		pb.Stop()
		fmt.Printf("❌ %s\n", err)
		os.Exit(1)
	}
	pb.Increment()

	if graph.Len() == 0 {
		fmt.Printf("❌ Empty DAG generated for model: %s\n", modelName)
		os.Exit(1)
	}

	return graph
}

func executeGraph(graph *fs.Graph) {
	pb := utils.NewProgressBar("🚀 Executing DAG", graph.Len())
	defer pb.Stop()

	graph.Execute(func(file *fs.File) {
		if file.Type == fs.ModelFile {
			if err := bigquery.Run(file); err != nil {
				pb.Stop()
				fmt.Printf("❌ %s\n", err)
				os.Exit(1)
			}
		}

		pb.Increment()
	}, utils.NumberWorkers)
}
