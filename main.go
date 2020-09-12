package main

import (
	"fmt"
	"log"

	"ddbt/fs"
	"ddbt/jinja"
)

func main() {
	fileSystem, err := fs.ReadFileSystem()
	if err != nil {
		log.Fatalf("Unable to read filesystem: %s", err)
	}

	//model := fileSystem.Model("")
	//if model != nil {
	//	fmt.Printf("Processing %s\n", model.Name)
	//
	//	body, err := jinja.Parse(model)
	//	if err != nil {
	//		log.Fatalf("Unable to parse model %s : %s", model.Name, err)
	//	}
	//
	//	fmt.Println(body)
	//}

	for _, macro := range fileSystem.Macros() {
		fmt.Printf("Processing %s\n", macro.Name)

		_, err := jinja.Parse(macro)
		if err != nil {
			log.Fatalf("Unable to parse macro %s : %s", macro.Name, err)
		}
	}

	for _, model := range fileSystem.Models() {
		fmt.Printf("Processing %s\n", model.Name)

		if _, err := jinja.Parse(model); err != nil {
			log.Fatalf("Unable to parse model %s : %s", model.Name, err)
		}
	}
}
