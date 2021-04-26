package cmd

import (
	"ddbt/properties"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestAddMissingColumnsToSchema(t *testing.T) {
	allColumns := []string{
		"column_a",
		"column_b",
		"column_c",
		"column_d",
		"column_e",
	}

	originalSchemaModel := generateNewSchemaModel("test_model", allColumns[:3])
	fullSchemaModel := generateNewSchemaModel("test_model", allColumns)

	addMissingColumnsToSchema(originalSchemaModel, allColumns)
	assert.Equal(t, originalSchemaModel, fullSchemaModel)
	require.Len(t, originalSchemaModel.Columns, len(allColumns), "Wrong number of columns in result")
}

func TestRemoveOutdatedColumnsFromSchema(t *testing.T) {
	allColumns := []string{
		"column_a",
		"column_b",
		"column_c",
		"column_d",
		"column_e",
	}
	updatedColumns := []string{
		"column_a",
		"column_d",
	}
	fullSchemaModel := generateNewSchemaModel("test_model", allColumns)
	updatedSchemaModel := generateNewSchemaModel("test_model", updatedColumns)

	removeOutdatedColumnsFromSchema(fullSchemaModel, updatedColumns)
	assert.Equal(t, updatedSchemaModel, fullSchemaModel)
	require.Len(t, fullSchemaModel.Columns, 2, "Wrong number of columns in result")
}

func TestParseExistingYMLSchema(t *testing.T) {
	schemaYML := `version: 2
models:
  - name: target_model
    columns:
      - name: column_a
        description: '{{ doc("staff_user_id") }}'
      - name: column_b
        description: "{{ doc('preferred_name') }}"
      - name: hibob_id
        description: '{{ doc("hibob_id") }}'
    `
	schema := &properties.File{}
	require.NoError(t, schema.Unmarshal([]byte(schemaYML)), "Unable to parse schema YAML")

	for _, model := range schema.Models {
		fmt.Printf("%+v\n", *model)
	}
	// fmt.Printf("%+v\n", schema.Models)
	// _ = `version: 2
	// models:
	//   - name: target_model
	// 	columns:
	// 		- name: staff_user_id
	// 			description: '{{ doc('staff_user_id') }}'
	// 			tests:
	// 			- unique
	// 			- not_null
	// 		- name: name
	// 			description: '{{ doc(''preferred_name'') }}'
	// 			tests: []
	// `
	marshalledYML, err := yaml.Marshal(schema)
	require.NoError(t, err, "Unable to marshall schema YAML")
	fmt.Println(string(marshalledYML))
}
