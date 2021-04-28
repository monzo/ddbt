package cmd

import (
	"context"
	"ddbt/bigquery"
	"ddbt/config"
	"ddbt/fs"
	schemaTestMacros "ddbt/schemaTestMacros"
	"ddbt/utils"
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"strconv"
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

		// not_null := schemaTestMacro.Test_not_null_macro("agents", "test")
		// fmt.Println(not_null)
		// unique := schemaTestMacro.Test_unique_macro("agents", "test")
		// fmt.Println(unique)

		// User prompt to make sure full table has been run in dev
		// Read table (using similar methods from schema-gen)
		// Apply test file to each column in BQ table -> evaluate result
		// Where test passes, suggest test (user prompt)
		// Write the test to the schema

		os.Exit(1)

	},
}

func generateTestsForModelsGraph(graph *fs.Graph) error {
	pb := utils.NewProgressBar("🖨️ Generating tests for models in graph", graph.Len())
	pb.Stop()

	ctx, cancel := context.WithCancel(context.Background())

	return graph.Execute(func(file *fs.File) error {
		if file.Type == fs.ModelFile {
			if err := generateTestsForModel(ctx, file); err != nil {
				pb.Stop()

				if err != context.Canceled {
					fmt.Printf("❌ %s\n", err)
				}

				cancel()
				return err
			}
		}

		// pb.Increment()
		return nil
	}, config.NumberThreads(), pb)

}

// generateTestsForModel generates tests for model and writes yml schema for modelName.
func generateTestsForModel(ctx context.Context, model *fs.File) error {
	target, err := model.GetTarget()
	if err != nil {
		fmt.Println("could not get target for schema")
		return err
	}
	fmt.Println("\n🎯 Target for retrieving schema:", target.ProjectID+"."+target.DataSet)

	// retrieve columns from BigQuery
	bqColumns, err := getColumnsForModel(ctx, model.Name, target)
	if err != nil {
		fmt.Println("Could not retrieve schema")
		return err
	}
	fmt.Println("✅ BQ Schema retrieved. Number of columns in BQ table:", len(bqColumns))

	// iterate through functions which return test sql and definition
	testFuncs := []func(string, string, string, string) (string, string){
		schemaTestMacros.Test_not_null_macro,
		schemaTestMacros.Test_unique_macro,
	}

	testColumnQueries := make(map[string]map[string]schemaTest)
	for _, col := range bqColumns {
		testsQueries := make(map[string]schemaTest)
		for _, test := range testFuncs {
			testQuery, testName := test(target.ProjectID, target.DataSet, model.Name, col)
			testsQueries[testName] = schemaTest{
				Query:       testQuery,
				QueryResult: false,
			}
		}
		testColumnQueries[col] = testsQueries
	}

	client, err := bigquery.GetClientFor(target.ProjectID)
	if err != nil {
		return err
	}

	//query := `select count(*)
	//from monzo-analytics-dev.dbt_ibrahimfaruqi_dev.support_identity_verification where support_idv_id is null`

	for col, tests := range testColumnQueries {
		for test, testQuery := range tests {
			q := client.Query(testQuery.Query)
			result, err := q.Read(ctx)
			if err != nil {
				return err
			}

			r, err := bigquery.ExtractBigqueryResult(result)
			if err != nil {
				return err
			}

			res := r[0][0]

			queryResult := int(res)
			if queryResult == 0 {
				fmt.Printf("Column %s is %s\n", col, test)
				testColumnQueries[col][test] = schemaTest{
					Query:       testQuery.Query,
					QueryResult: true,
				}

			}
		}
	}


	//result, err := bigquery.RunQuery(ctx, model.Name, query, target)
	//if err != nil {
	//	return err
	//}
	//fmt.Println("results:", result)

	return nil

	// // create schema file
	// ymlPath, schemaFile := generateEmptySchemaFile(model)
	// var schemaModel *properties.Model

	// if model.Schema == nil {
	// 	fmt.Println("\n🔍 " + model.Name + " schema file not found.. 🌱 Generating new schema file")
	// 	schemaModel = generateNewSchemaModel(model.Name, bqColumns)
	// } else {
	// 	fmt.Println("\n🔍 " + model.Name + " schema file found.. 🛠  Updating schema file")
	// 	// set working schema model to current schema model
	// 	schemaModel = model.Schema
	// 	// add and remove columns in-place
	// 	addMissingColumnsToSchema(schemaModel, bqColumns)
	// 	removeOutdatedColumnsFromSchema(schemaModel, bqColumns)
	// }

	// schemaFile.Models = properties.Models{schemaModel}
	// err = schemaFile.WriteToFile(ymlPath)
	// if err != nil {
	// 	fmt.Println("Error writing YML to file in path")
	// 	return err
	// }
	// fmt.Println("\n✅ " + model.Name + "schema successfully updated at path: " + ymlPath)

	// return nil
}
