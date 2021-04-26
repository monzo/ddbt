package cmd

import (
	"context"
	"ddbt/bigquery"
	"ddbt/config"
	"ddbt/fs"
	"ddbt/properties"
	"ddbt/utils"

	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(schemaGenCmd)
	addModelsFlag(schemaGenCmd)
}

var schemaGenCmd = &cobra.Command{
	Use:               "schema-gen [model name]",
	Short:             "Generates the YML schema file for a given model",
	Args:              cobra.RangeArgs(0, 1),
	ValidArgsFunction: completeModelFn,
	Run: func(cmd *cobra.Command, args []string) {
		switch {
		case len(args) == 0 && len(ModelFilters) == 0:
			fmt.Println("Please specify model with schema-gen -m model-name")
			os.Exit(1)
		case len(args) == 1 && len(ModelFilters) > 0:
			fmt.Println("Please specify model with either schema-gen model-name or schema-gen -m model-name but not both")
			os.Exit(1)
		case len(args) == 1:
			// This will actually allow something weird like
			// ddbt schema-gen +model+
			ModelFilters = append(ModelFilters, args[0])
		}

		// Build a graph from the given filter.
		fileSystem, _ := compileAllModels()
		for k, v := range fileSystem.Docs {
			fmt.Println("key:", k, "value:", v)
		}

		// graph := buildGraph(fileSystem, ModelFilters)

		// // Generate schema for every file in the graph concurrently.
		// if err := generateSchemaForGraph(graph); err != nil {
		// 	fmt.Printf("❌ %s\n", err)
		// 	os.Exit(1)
		// }
		os.Exit(1)

	},
}

func generateSchemaForGraph(graph *fs.Graph) error {
	pb := utils.NewProgressBar("🖨️ Generating schemas", graph.Len())
	defer pb.Stop()

	ctx, cancel := context.WithCancel(context.Background())

	return graph.Execute(func(file *fs.File) error {
		if file.Type == fs.ModelFile {
			if err := generateSchemaForModel(ctx, file); err != nil {
				pb.Stop()

				if err != context.Canceled {
					fmt.Printf("❌ %s\n", err)
				}

				cancel()
				return err
			}
		}

		pb.Increment()
		return nil
	}, config.NumberThreads(), pb)
}

// generateSchemaForModel generates a schema and writes yml for modelName.
func generateSchemaForModel(ctx context.Context, model *fs.File) error {
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

	// create schema file
	ymlPath, schemaFile := generateEmptySchemaFile(model)
	var schemaModel *properties.Model

	if model.Schema == nil {
		fmt.Println("\n🔍 " + model.Name + " schema file not found.. 🌱 Generating new schema file")
		schemaModel = generateNewSchemaModel(model.Name, bqColumns)
	} else {
		fmt.Println("\n🔍 " + model.Name + " schema file found.. 🛠  Updating schema file")
		// set working schema model to current schema model
		schemaModel = model.Schema
		// add and remove columns in-place
		addMissingColumnsToSchema(schemaModel, bqColumns)
		removeOutdatedColumnsFromSchema(schemaModel, bqColumns)
	}

	schemaFile.Models = properties.Models{schemaModel}
	err = schemaFile.WriteToFile(ymlPath)
	if err != nil {
		fmt.Println("Error writing YML to file in path")
		return err
	}
	fmt.Println("\n✅ " + model.Name + "schema successfully updated at path: " + ymlPath)
	return nil
}

func getColumnsForModel(ctx context.Context, modelName string, target *config.Target) ([]string, error) {
	schema, err := bigquery.GetColumnsFromTableWithContext(ctx, modelName, target)
	if err != nil {
		return nil, err
	}

	columns := []string{}
	for _, fieldSchema := range schema {
		column := fmt.Sprintf("%v", fieldSchema.Name)
		columns = append(columns, column)
	}
	return columns, nil
}

// generate an empty schema file which will be populated according to existing yml schemas and the bigquery schema.
// Returns the local path for the yml file and the yml file struct
func generateEmptySchemaFile(model *fs.File) (ymlPath string, schemaFile properties.File) {
	ymlPath = strings.Replace(model.Path, ".sql", ".yml", 1)
	schemaFile = properties.File{}
	schemaFile.Version = properties.FileVersion
	return ymlPath, schemaFile
}

// generate a new schema model for the provided model name and bqcolumns
// this is used when then is no existing model
func generateNewSchemaModel(modelName string, bqColumns []string) *properties.Model {
	schemaModel := &properties.Model{}
	schemaModel.Name = modelName
	schemaModel.Description = "Please fill this in with a useful description.."
	schemaModel.Columns = make([]properties.Column, 0, len(bqColumns))
	for _, bqCol := range bqColumns {
		column := properties.Column{}
		column.Name = bqCol
		schemaModel.Columns = append(schemaModel.Columns, column)
	}

	return schemaModel
}

// check if bq column is in schema (add missing)
func addMissingColumnsToSchema(schemaModel *properties.Model, bqColumns []string) {
	columnsAdded := []string{}

	schemaColumnMap := make(map[string]bool)
	for _, schemaCol := range schemaModel.Columns {
		schemaColumnMap[schemaCol.Name] = true
	}

	for _, bqCol := range bqColumns {
		if _, exists := schemaColumnMap[bqCol]; !exists {
			column := properties.Column{}
			column.Name = bqCol
			schemaModel.Columns = append(schemaModel.Columns, column)
			columnsAdded = append(columnsAdded, bqCol)
		}
	}
	fmt.Println("➕ Columns added to Schema (from BQ table):", columnsAdded)
}

// check if schema column in bq (remove missing)
func removeOutdatedColumnsFromSchema(schemaModel *properties.Model, bqColumns []string) {
	columnsRemoved := []string{}
	columnsKept := properties.Columns{}

	bqColumnMap := make(map[string]bool)
	for _, bqCol := range bqColumns {
		bqColumnMap[bqCol] = true
	}

	for _, schemaCol := range schemaModel.Columns {
		if _, exists := bqColumnMap[schemaCol.Name]; !exists {
			columnsRemoved = append(columnsRemoved, schemaCol.Name)
		} else {
			columnsKept = append(columnsKept, schemaCol)
		}
	}
	schemaModel.Columns = columnsKept
	fmt.Println("➖ Columns removed from Schema (no longer in BQ table):", columnsRemoved)
}
