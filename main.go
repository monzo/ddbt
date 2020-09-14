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
			_, err := compiler.CompileModel(file, gc)
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
			_, err := compiler.CompileModel(file, gc)
			if err != nil {
				pb.Stop()
				fmt.Printf("❌ Unable to compile %s %s: %s\n", file.Type, file.Name, err)
				os.Exit(1)
			}
			pb.Increment()
		},
	)
}
