package main

import (
	"fmt"
	"log"
	"os"

	"ddbt/compiler"
	"ddbt/fs"
	"ddbt/utils"
)

func main() {
	fileSystem, err := fs.ReadFileSystem()
	if err != nil {
		log.Fatalf("Unable to read filesystem: %s", err)
	}

	parseFiles(fileSystem)
	gc := compiler.NewGlobalContext(fileSystem)
	compileMacros(fileSystem, gc)
	compileFiles(fileSystem, gc)

	if len(os.Args) > 1 {
		modelName := os.Args[1]

		executionOrder := buildGraph(fileSystem, modelName)

		for depth, files := range executionOrder {
			fmt.Printf("\nExecution Batch %d\n", depth+1)

			for _, file := range files {
				fmt.Printf("\t- %s\n", file.Name)
			}
		}
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

func buildGraph(fileSystem *fs.FileSystem, modelName string) [][]*fs.File {
	pb := utils.NewProgressBar("🕸 Building DAG", 1)
	defer pb.Stop()

	model := fileSystem.Model(modelName)
	if model == nil {
		pb.Stop()
		fmt.Printf("❌ Unable to find model: %s\n", modelName)
		os.Exit(1)
	}

	// The selected model always has a depth of 0
	modelDepths := make(map[*fs.File]int)

	// Check the upstreams
	largestDepth, err := model.BuildUpstreamDag(0, modelDepths, make(map[*fs.File]struct{}))
	if err != nil {
		pb.Stop()
		fmt.Printf("❌ %s\n", err)
		os.Exit(1)
	}

	// Build execution order
	largestDepth += 1 // (account for the fact our range is `0 <= y <= largestDepth`
	executionOrder := make([][]*fs.File, largestDepth)
	for i := 0; i < largestDepth; i++ {
		executionOrder[i] = make([]*fs.File, 0)
	}
	for file, depth := range modelDepths {
		executionOrder[largestDepth-depth-1] = append(executionOrder[largestDepth-depth-1], file)
	}

	pb.Increment()

	return executionOrder
}
