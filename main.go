package main

import (
	"fmt"
	"log"
	"os"

	"ddbt/compiler"
	"ddbt/fs"
	"ddbt/jinja"
	"ddbt/utils"
)

func main() {
	fileSystem, err := fs.ReadFileSystem()
	if err != nil {
		log.Fatalf("Unable to read filesystem: %s", err)
	}

	parseFiles(fileSystem)
	compileFiles(fileSystem)
}

func parseFiles(fileSystem *fs.FileSystem) {
	pb := utils.NewProgressBar("📜 Reading & Parsing Files", fileSystem.NumberFiles())
	defer pb.Stop()

	utils.ProcessFiles(
		fileSystem.AllFiles(),
		func(file *fs.File) {
			if err := parseFile(file); err != nil {
				pb.Stop()
				fmt.Printf("❌ Unable to parse %s %s: %s\n", file.Type, file.Name, err)
				os.Exit(1)
			}
			pb.Increment()
		},
	)
}

func parseFile(file *fs.File) error {
	syntaxTree, err := jinja.Parse(file)
	if err != nil {
		return err
	}

	file.SyntaxTree = syntaxTree
	return nil
}

func compileFiles(fileSystem *fs.FileSystem) {
	pb := utils.NewProgressBar("📝 Compiling Models", len(fileSystem.Models()))
	defer pb.Stop()

	utils.ProcessFiles(
		fileSystem.Models(),
		func(file *fs.File) {
			_, err := compiler.CompileModel(file)
			if err != nil {
				pb.Stop()
				fmt.Printf("❌ Unable to compile %s %s: %s\n", file.Type, file.Name, err)
				os.Exit(1)
			}
			pb.Increment()
		},
	)
}
