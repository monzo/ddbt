package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	schemaTestMacro "ddbt/schema_test_macros"
)

// Predefined tests we want to check for:
// - Uniquness
// - Not null

// STRETCH:
// Parallelise running tests for each column)
// Parse macro files
// Test with value inputs e.g. accepted values

func init() {
	rootCmd.AddCommand(testGenCmd)
	addModelsFlag(testGenCmd)
}

type TestMacro struct {
	Name     string
	Filepath string
	Contents string
}

var testGenCmd = &cobra.Command{
	Use:               "test-gen [model name]",
	Short:             "Suggests tests to add to the YML schema file for a given model",
	Args:              cobra.RangeArgs(0, 1),
	ValidArgsFunction: completeModelFn,
	Run: func(cmd *cobra.Command, args []string) {
		switch {
		case len(args) == 0 && len(ModelFilters) == 0:
			fmt.Println("Please specify model with test-gen -m model-name")
			os.Exit(1)
		case len(args) == 1 && len(ModelFilters) > 0:
			fmt.Println("Please specify model with either test-gen model-name or test-gen -m model-name but not both")
			os.Exit(1)
		case len(args) == 1:
			// This will actually allow something weird like
			// ddbt schema-gen +model+
			ModelFilters = append(ModelFilters, args[0])
		}

		// Build a graph from the given filter.
		// fileSystem, _ := compileAllModels()

		// graph := buildGraph(fileSystem, ModelFilters)

		not_null := schemaTestMacro.Test_not_null_macro()
		fmt.Println(not_null)
		// User prompt to make sure full table has been run in dev
		// Read table (using similar methods from schema-gen)
		// Apply test file to each column in BQ table -> evaluate result
		// Where test passes, suggest test (user prompt)
		// Write the test to the schema

		os.Exit(1)

	},
}

// // Read SQL file in as string
// func readSchemaTestMacros() {

// 	err := filepath.Walk(".",
// 		func(path string, info os.FileInfo, err error) error {
// 			if err != nil {
// 				return err
// 			}
// 			fmt.Println(path, info.Size())
// 			return nil
// 		})
// 	if err != nil {
// 		log.Println(err)
// 	}

// 	// var files []schemaTestMacro
// 	// _ = filepath.Walk("../schema_test_macros", func(filePath string, info os.FileInfo, err error) error {
// 	// 	fileContents, _ := ioutil.ReadFile(filePath)
// 	// 	sqlMacro := string(fileContents)

// 	// 	splitPath := strings.Split(filePath, "/")
// 	// 	fileName := splitPath[len(splitPath)-1]

// 	// 	name := strings.TrimSuffix(fileName, filepath.Ext(fileName))

// 	// 	files = append(files, schemaTestMacro{
// 	// 		Name:     name,
// 	// 		Filepath: filePath,
// 	// 		Contents: sqlMacro,
// 	// 	})
// 	// 	return nil
// 	// },
// 	// )
// 	// return files
// }

// func readSqlFile(filePath string, macrosMap map[string]schemaTestMacro) map[string]schemaTestMacro {
// 	fileContents, _ := ioutil.ReadFile(filePath)
// 	sqlMacro := string(fileContents)

// 	splitPath := strings.Split(filePath, "/")
// 	fileName := splitPath[len(splitPath)-1]

// 	name := strings.TrimSuffix(fileName, filepath.Ext(fileName))

// 	macrosMap[fileName] = schemaTestMacro{
// 		Name:     name,
// 		Filepath: filePath,
// 		Contents: sqlMacro,
// 	}
// 	return macrosMap
// }
