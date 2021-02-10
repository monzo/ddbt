package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
