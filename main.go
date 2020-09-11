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

	macro := fileSystem.Macro("test_x_pct_not_null_in_last_num_days")
	if err := jinja.Parse(macro); err != nil {
		log.Fatalf("Unable to parse %s: %s", macro.Name, err)
	}

	for _, macro := range fileSystem.Macros() {
		fmt.Printf("Processing %s\n", macro.Name)

		if err := jinja.Parse(macro); err != nil {
			log.Fatalf("Unable to parse %s: %s", macro.Name, err)
		}
	}

	//for _, model := range fileSystem.Models() {
	//	fmt.Printf("Processing %s\n", model.Name)
	//
	//	if _, err := jinja.LexFile(model.Path); err != nil {
	//		log.Fatalf("Unable to lex %s: %s", model.Name, err)
	//	}
	//
	//	return
	//}
}
