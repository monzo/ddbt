package cmd

import (
	"context"
	"ddbt/bigquery"
	"ddbt/config"
	"ddbt/fs"
	"ddbt/properties"
	schemaTestMacros "ddbt/schemaTestMacros"
	"ddbt/utils"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/spf13/cobra"
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

type TestSuggestions struct {
	mu          sync.Mutex
	suggestions map[string]map[string][]string
}

func (d *TestSuggestions) SetSuggestion(modelName string, testSuggestions map[string][]string) {
	d.mu.Lock()
	d.suggestions[modelName] = testSuggestions
	d.mu.Unlock()
}

func (d *TestSuggestions) Init() {
	d.mu.Lock()
	d.suggestions = make(map[string]map[string][]string)
	d.mu.Unlock()
}

func (d *TestSuggestions) Value() (suggestions map[string]map[string][]string) {
	d.mu.Lock()
	suggestions = d.suggestions
	d.mu.Unlock()
	return
}

type schemaTest struct {
	Query       string
	QueryResult bool
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
		fileSystem, _ := compileAllModels()
		graph := buildGraph(fileSystem, ModelFilters)

		// Generate schema for every file in the graph concurrently.
		if err := generateTestsForModelsGraph(graph); err != nil {
			fmt.Printf("❌ %s\n", err)
			os.Exit(1)
		}

		os.Exit(1)

	},
}

func generateTestsForModelsGraph(graph *fs.Graph) error {
	pb := utils.NewProgressBar("🖨 Generating tests for models in graph", graph.Len())

	ctx, cancel := context.WithCancel(context.Background())
	var testSugs TestSuggestions
	testSugs.Init()

	err := graph.Execute(func(file *fs.File) error {
		if file.Type == fs.ModelFile {
			testSuggestions, err := generateTestsForModel(ctx, file)
			if err != nil {
				pb.Stop()
				if err != context.Canceled {
					fmt.Printf("❌ %s\n", err)
				}
				cancel()
				return err
			}
			testSugs.SetSuggestion(file.Name, testSuggestions)
		}

		pb.Increment()
		return nil
	}, config.NumberThreads(), pb)

	if err != nil {
		return err
	}
	pb.Stop()

	err = userPromptTests(graph, testSugs.suggestions)
	if err != nil {
		return err
	}

	return nil
}

// generateTestsForModel generates tests for model and writes yml schema for modelName.
func generateTestsForModel(ctx context.Context, file *fs.File) (map[string][]string, error) {
	target, err := file.GetTarget()
	if err != nil {
		fmt.Println("could not get target for schema")
		return nil, err
	}
	fmt.Println("\n🎯 Target for retrieving schema:", target.ProjectID+"."+target.DataSet)

	// retrieve columns from BigQuery
	bqColumns, err := getColumnsForModel(ctx, file.Name, target)
	if err != nil {
		fmt.Println("Could not retrieve schema")
		return nil, err
	}
	fmt.Println("✅ BQ Schema retrieved. Number of columns in BQ table:", len(bqColumns))

	// iterate through functions which return test sql and definition
	testFuncs := []func(string, string, string, string) (string, string){
		schemaTestMacros.Test_not_null_macro,
		schemaTestMacros.Test_unique_macro,
	}

	allTestQueries := make(map[string]map[string]string)

	for _, col := range bqColumns {
		testsQueries := make(map[string]string)
		for _, test := range testFuncs {
			testQuery, testName := test(target.ProjectID, target.DataSet, file.Name, col)
			testsQueries[testName] = testQuery
		}
		allTestQueries[col] = testsQueries
	}

	passedTestQueries := make(map[string][]string)

	for col, tests := range allTestQueries {
		for test, testQuery := range tests {

			var rows uint64
			var results [][]bigquery.Value
			results, _, err = bigquery.GetRows(ctx, testQuery, target)

			if err == nil {
				if len(results) != 1 {
					err = errors.New(fmt.Sprintf("a schema test should only return 1 row, got %d", len(results)))
				} else if len(results[0]) != 1 {
					err = errors.New(fmt.Sprintf("a schema test should only return 1 column, got %d", len(results[0])))
				} else {
					rows, err = bigquery.ValueAsUint64(results[0][0])
				}
			}
			if err != nil {
				return nil, err
			}

			if rows == 0 {
				if _, contains := passedTestQueries[col]; contains {
					passedTestQueries[col] = append(passedTestQueries[col], test)
				} else {
					passedTestQueries[col] = []string{test}
				}
			}
		}
	}
	updateSchemaFile(passedTestQueries, file)

	return passedTestQueries, nil
}

func updateSchemaFile(passedTestQueries map[string][]string, model *fs.File) {
	updatedColumns := model.Schema.Columns
	for colIndex, column := range model.Schema.Columns {
		if _, exists := passedTestQueries[column.Name]; exists {

			// search for test in existing tests
			for _, test := range passedTestQueries[column.Name] {
				testFound := false
				for _, existingTest := range column.Tests {
					if existingTest.Name == test {
						testFound = true
						break
					}
				}
				if !testFound {
					column.Tests = append(column.Tests, &properties.Test{
						Name: test,
					})
				}
			}
		}
		updatedColumns[colIndex] = column
	}
	model.Schema.Columns = updatedColumns
}

func userPromptTests(graph *fs.Graph, testSugsMap map[string]map[string][]string) error {
	if len(testSugsMap) > 0 {
		fmt.Println("\n🧪 Valid tests found for the following models: ")
		for model, columnTests := range testSugsMap {
			fmt.Println("\n🧬 Model:", model)
			for column, tests := range columnTests {
				fmt.Println("🏛 Column:", column)
				testPrint := strings.Join(tests[:], "\n  - ")
				fmt.Println("  -", testPrint)
			}
		}
		fmt.Println("\n❔ Would you like to add these tests to the schema (y/N)?")

		var userPrompt string
		fmt.Scanln(&userPrompt)

		if userPrompt == "y" {
			for file, _ := range graph.ListNodes() {
				if _, contains := testSugsMap[file.Name]; contains {
					ymlPath, schemaFile := generateEmptySchemaFile(file)
					schemaModel := file.Schema
					schemaFile.Models = properties.Models{schemaModel}
					err := schemaFile.WriteToFile(ymlPath)
					if err != nil {
						fmt.Println("Error writing YML to file in path")
						return err
					}
				}
			}
			fmt.Println("✅ Tests added to schema files")
		}
	}
	return nil
}
