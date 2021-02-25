package cmd

import (
	"ddbt/bigquery"
	"ddbt/config"

	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/atotto/clipboard"
)

// map bigquery data types to looker data types
var mapBqToLookerDtypes map[string]string = map[string]string{
	    "INTEGER": "number",
	    "FLOAT":   "number",
	    "NUMERIC": "number",
	    "BOOLEAN": "yesno",
	    "STRING": "string",
	    "TIMESTAMP": "time",
	    "DATETIME": "time",
	    "DATE":  "time",
	    "TIME": "time",
	    "BOOL": "yesno",
	    "ARRAY": "string",
	    "GEOGRAPHY": "string",
	}

// specify looker timeframes for datetime/date/time variable data types
const timeBlock string = `timeframes: [
  raw,
  time,
  date,
  week,
  month,
  quarter,
  year
]
`

// specify supportal linking for user_id
const userIdLink string = `link: {
  label: "Supportal"
  url: "https://supportal.tools.prod.prod-ffs.io/users/{{ value }}"
}
`

func init() {
	rootCmd.AddCommand(lookmlGenCmd)
}

var lookmlGenCmd = &cobra.Command{
	Use:               "lookml-gen [model name]",
	Short:             "Generates the .view.lkml file for a given model",
	Args:              cobra.ExactValidArgs(1),
	ValidArgsFunction: completeModelFn,
	Run: func(cmd *cobra.Command, args []string) {
		modelName := args[0]

		// get filesystem, model and target
		fileSystem, _ := compileAllModels()
		model := fileSystem.Model(modelName)

		target, err := model.GetTarget()
		if err != nil {
			fmt.Println("Could not get target for schema")
			os.Exit(1)
		}
		fmt.Println("\n🎯 Target for retrieving schema:", target.ProjectID+"."+target.DataSet)

		// generate lookml view
		err = generateNewLookmlView(modelName, target)

		if err != nil {
			fmt.Println("😒 Something went wrong at lookml view generation: ", err)
			os.Exit(1)
		}

	},
}

func getColumnsForModelWithDtypes(modelName string, target *config.Target) (columns []string, dtypes []string, err error) {
	schema, err := bigquery.GetColumnsFromTable(modelName, target)
	if err != nil {
		return nil, nil, err
	}

	// itereate over fields, record field names and data types
	for _, fieldSchema := range schema {
		column := fmt.Sprintf("%v", fieldSchema.Name)
		dtype := fmt.Sprintf("%v", fieldSchema.Type)
		columns = append(columns, column)
		dtypes = append(dtypes, dtype)

	}
	return columns, dtypes, nil
}

func generateNewLookmlView(modelName string, target *config.Target) (error) {
	bqColumns, bqDtypes, err := getColumnsForModelWithDtypes(modelName, target)
	if err != nil {
		fmt.Println("Could not retrieve schema from BigQuery")
		os.Exit(1)
	}

	// initialise lookml view head
	lookmlView := "view: " + modelName +" {\n\n"
	lookmlView += "sql_table_name: `" + target.ProjectID + "." + target.DataSet + "." + modelName + "` ;;\n"

	// add dimensions and appropriate blocks for each field
	for i := 0; i < len(bqColumns); i ++ {
		colName := bqColumns[i]
		colDtype := mapBqToLookerDtypes[bqDtypes[i]]
		newBlock := "\n"
		
		if colDtype == "date_time" || colDtype == "date" || colDtype == "time"{
			newBlock += "dimension_group: " + colName + " {\n"
			newBlock += "type: " + colDtype + "\n"
			newBlock += timeBlock
		} else {
			newBlock += "dimension: " + colName + " {\n"
			newBlock += "type: " + colDtype + "\n"
		}

		if colName == "user_id" {
			newBlock += userIdLink
		}
		
		newBlock += "sql: ${TABLE}." + colName + " ;;\n}\n"

		lookmlView += newBlock
	}

	// add closing curly bracket and copy to clipboard
	lookmlView += "}"

	clipboard.WriteAll(lookmlView)
	fmt.Println("\n✅ LookML view for " + modelName + " has been copied to your clipboard!")

	return nil
}

