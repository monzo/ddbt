package main

import (
	"log"

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

	pb := utils.NewProgressBar("📝 Compiling Models", len(fileSystem.Models()))
	pb.Start()
}

func parseFiles(fileSystem *fs.FileSystem) {
	pb := utils.NewProgressBar("📜 Reading & Parsing Files", fileSystem.NumberFiles())
	defer pb.Stop()

	utils.ProcessFiles(
		fileSystem.AllFiles(),
		func(file *fs.File) {
			_, err := jinja.Parse(file)
			if err != nil {
				log.Fatalf("Unable to parse %s : %s", file.Name, err)
			}

			pb.Increment()
		},
	)
}
