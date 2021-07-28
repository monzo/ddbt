package cmd

import (
	"context"
	"ddbt/bigquery"
	"ddbt/config"
	"ddbt/fs"
	"ddbt/properties"
	schemaTestMacros "ddbt/schemaTestMacros"
	"ddbt/utils"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/spf13/cobra"
)

// TODO:
// only output suggestions to terminal for new tests
// Parse macro files
// Test with value inputs e.g. accepted values

func init() {
	rootCmd.AddCommand(testGenCmd)
	addModelsFlag(testGenCmd)
}

type ColumnTestQuery struct {
	Column    string
	TestName  string
	TestQuery string
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
			// ddbt test-gen +model+
			ModelFilters = append(ModelFilters, args[0])
		}

		fmt.Println(`ℹ️  test-gen requires models without sampling to be accurate
❓  Have you run the provided models with sampling off? (y/N)`)
		var userPrompt string
		fmt.Scanln(&userPrompt)
		if userPrompt != "y" {
			fmt.Println("❌ Please run your model with sampling and then use test-gen")
			os.Exit(1)
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
		schemaTestMacros.TestNotNullMacro,
		schemaTestMacros.TestUniqueMacro,
	}

	var allTestQueries []ColumnTestQuery
	for _, col := range bqColumns {
		for _, test := range testFuncs {
			testQuery, testName := test(target.ProjectID, target.DataSet, file.Name, col)
			allTestQueries = append(allTestQueries, ColumnTestQuery{
				Column:    col,
				TestName:  testName,
				TestQuery: testQuery,
			})
		}
	}

	passedTestQueries, err := runQueriesParallel(ctx, target, allTestQueries)
	if err != nil {
		return nil, err
	}
	updateSchemaFile(passedTestQueries, file)

	return passedTestQueries, nil
}

func runQueriesParallel(ctx context.Context, target *config.Target, allTestQueries []ColumnTestQuery) (map[string][]string, error) {
	// number of parallel query runners
	numQueryRunners := 100

	queries := make(chan ColumnTestQuery)
	go func() {
		for _, q := range allTestQueries {
			queries <- q
		}
		close(queries)
	}()

	out := make(chan ColumnTestQuery, len(allTestQueries))
	errs := make(chan error, len(allTestQueries))
	wg := sync.WaitGroup{}

	wg.Add(numQueryRunners)
	for i := 0; i < numQueryRunners; i++ {
		go func(i int) {
			defer wg.Done()
			for query := range queries {
				evaluateTestQuery(ctx, target, query, out, errs, i)
			}
		}(i)
	}

	go func() {
		wg.Wait()
		close(out)
		close(errs)
	}()

	if err := <-errs; err != nil {
		// there is at least one error, but we ignore the rest
		return nil, err
	}

	passedTestQueries := make(map[string][]string)
	for passedTestQuery := range out {
		if _, contains := passedTestQueries[passedTestQuery.Column]; contains {
			passedTestQueries[passedTestQuery.Column] = append(passedTestQueries[passedTestQuery.Column], passedTestQuery.TestName)
		} else {
			passedTestQueries[passedTestQuery.Column] = []string{passedTestQuery.TestName}
		}
	}

	return passedTestQueries, nil
}

func evaluateTestQuery(ctx context.Context, target *config.Target, ctq ColumnTestQuery, out chan ColumnTestQuery,
	errs chan error, workerIndex int) {
	results, _, err := bigquery.GetRows(ctx, ctq.TestQuery, target)

	if err == nil {
		if len(results) != 1 {
			errs <- fmt.Errorf(fmt.Sprintf(
				"a schema test should only return 1 row, got %d for %s test on column %s by worker %v",
				len(results), ctq.TestName, ctq.Column, workerIndex),
			)
		} else if len(results[0]) != 1 {
			errs <- fmt.Errorf(fmt.Sprintf(
				"a schema test should only return 1 column, got %d for %s test on column %s by worker %v",
				len(results), ctq.TestName, ctq.Column, workerIndex),
			)
		} else {
			rows, _ := bigquery.ValueAsUint64(results[0][0])
			if rows == 0 {
				out <- ctq
			}
		}
	}
	if err != nil {
		errs <- err
	}

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
			for file := range graph.ListNodes() {
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
