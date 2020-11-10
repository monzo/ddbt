package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ddbt/compiler"
	"ddbt/properties"
)

const testTableRef = "`unit_test_project`.`unit_test_dataset`.`target_model`"

// Helper function for testing schema tests
func assertTestSchema(t *testing.T, propertiesYaml string, expectedTestSQL string) {
	schema := &properties.File{}
	require.NoError(t, schema.Unmarshal([]byte(propertiesYaml)), "Unable to parse schema YAML")

	tests, err := schema.DefinedTests()
	require.NoError(t, err)
	require.Len(t, tests, 1, "Only expected 1 test to be generated")

	fileSystem, gc, _ := CompileFromRaw(t, "SELECT 1 as column_a")

	for testName, testContents := range tests {
		file, err := fileSystem.AddTestWithContents(testName, testContents)
		require.NoError(t, err, "Unable to add test file")
		require.NoError(t, compiler.ParseFile(file), "Unable to parse test file")
		require.NoError(t, compiler.CompileModel(file, gc, true), "Unable to compile test file")

		assert.Equal(t, expectedTestSQL, file.CompiledContents, "Compiled test doesn't match")
	}
}

func TestTestUniqueMacro(t *testing.T) {
	assertTestSchema(t,
		`version: 2
models:
  - name: target_model
    columns:
      - name: column_a
        tests:
          - unique
      - name: column_b
`,
		`
	SELECT
	column_a AS value,
	COUNT(column_a) AS count
	
	FROM `+testTableRef+`
	
	GROUP BY column_a 
	
	HAVING COUNT(column_a) > 1
`,
	)
}

func TestTestNotNull(t *testing.T) {
	assertTestSchema(t,
		`version: 2
models:
  - name: target_model
    columns:
      - name: column_a
        tests:
          - not_null
      - name: column_b
`,
		`
	SELECT
	column_a AS value
	
	FROM `+testTableRef+`
	
	WHERE column_a IS NULL
`,
	)
}

func TestAcceptedValues(t *testing.T) {
	assertTestSchema(t,
		`version: 2
models:
  - name: target_model
    columns:
      - name: column_a
        tests:
          - accepted_values:
              values: ['foo', 'bar']
      - name: column_b
`,
		`
	SELECT
	column_a AS value
	
	FROM `+testTableRef+`

	WHERE column_a NOT IN (
		'foo', 'bar'
	)
`,
	)

	assertTestSchema(t,
		`version: 2
models:
  - name: target_model
    columns:
      - name: column_a
        tests:
          - accepted_values:
              values: ['foo', 'bar']
              quote: false
      - name: column_b
`,
		`
	SELECT
	column_a AS value
	
	FROM `+testTableRef+`

	WHERE column_a NOT IN (
		foo, bar
	)
`,
	)
}

func TestTestRelationships(t *testing.T) {
	assertTestSchema(t,
		`version: 2
models:
  - name: target_model
    columns:
      - name: column_a
        tests:
          - relationships:
              to: another_table
              field: id
      - name: column_b
`,
		`
	SELECT
	column_a AS value

	FROM `+testTableRef+` AS src

	LEFT JOIN another_table AS dest
	ON dest.id = src.column_a

	WHERE dest.id IS NULL AND src.column_a IS NOT NULL
`,
	)
}
