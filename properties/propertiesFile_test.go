package properties

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestPropertiesParse(t *testing.T) {
	yml := `version: 2
models:
- name: model_name
  description: random description
  docs:
    show: true
  meta:
    test: value
  tests:
  - unique:
      column_name: a || '-' || b
  columns:
  - name: column_name
    description: Test description
    tests:
    - not_null
    tags: ['some:tag', 'tag:2']
  - name: another_column
    description: |
      This is a multi line description!
    quote: true
    tests:
    - accepted_values:
        values:
        - foo
        - bar
        tags:
        - a
        - b
        - c
    - relationships:
        to: ref('another_model')
        field: id
    - unique:
        severity: warn
- name: another_model
macros:
- name: my_macro
  description: some macro which does stuff
  docs:
    show: false
  arguments:
  - name: test_arg
    type: string
    description: another description block
  - name: second_arg
    type: bool
    description: |
      Test description 2
`
	file := &File{}
	require.NoError(t, yaml.Unmarshal([]byte(yml), file))

	assert.Equal(t, 2, file.Version)
	require.Len(t, file.Models, 2, "Invalid number of models")

	assert.Equal(t, "model_name", file.Models[0].Name)
	assert.Equal(t, "random description", file.Models[0].Description)
	assert.Equal(t, true, *file.Models[0].Docs.Show)

	require.Len(t, file.Models[0].Tests, 1, "Not enough model level tests")
	assert.Equal(t, "unique", file.Models[0].Tests[0].Name)
	assert.Equal(t, "column_name", file.Models[0].Tests[0].Arguments[0].Name)
	assert.Equal(t, "a || '-' || b", file.Models[0].Tests[0].Arguments[0].Value)
	assert.Empty(t, file.Models[0].Tests[0].Tags)
	assert.Empty(t, file.Models[0].Tests[0].Severity)

	require.Len(t, file.Models[0].Columns, 2, "Not enough columns")

	// Test column 1
	assert.Equal(t, "column_name", file.Models[0].Columns[0].Name)
	assert.Equal(t, "Test description", file.Models[0].Columns[0].Description)
	assert.Equal(t, false, file.Models[0].Columns[0].Quote)

	require.Len(t, file.Models[0].Columns[0].Tests, 1, "Not enough tests on the first column")
	require.Equal(t, "not_null", file.Models[0].Columns[0].Tests[0].Name)
	assert.Empty(t, file.Models[0].Columns[0].Tests[0].Arguments)
	assert.Empty(t, file.Models[0].Columns[0].Tests[0].Tags)
	assert.Empty(t, file.Models[0].Columns[0].Tests[0].Severity)

	// Test column 2
	assert.Equal(t, "another_column", file.Models[0].Columns[1].Name)
	assert.Equal(t, "This is a multi line description!\n", file.Models[0].Columns[1].Description)
	assert.Equal(t, true, file.Models[0].Columns[1].Quote)
	require.Len(t, file.Models[0].Columns[1].Tests, 3, "Not enough tests on the second column")

	// Test column 2; test 1
	require.Equal(t, "accepted_values", file.Models[0].Columns[1].Tests[0].Name)
	assert.Equal(t, TestArguments{TestArgument{"values", []interface{}{"foo", "bar"}}}, file.Models[0].Columns[1].Tests[0].Arguments)
	assert.Equal(t, []string{"a", "b", "c"}, file.Models[0].Columns[1].Tests[0].Tags)
	assert.Empty(t, file.Models[0].Columns[1].Tests[0].Severity)

	// Test column 2; test 2
	require.Equal(t, "relationships", file.Models[0].Columns[1].Tests[1].Name)
	assert.Equal(t, TestArguments{TestArgument{"to", "ref('another_model')"}, TestArgument{"field", "id"}}, file.Models[0].Columns[1].Tests[1].Arguments)
	assert.Empty(t, file.Models[0].Columns[1].Tests[1].Tags)
	assert.Empty(t, file.Models[0].Columns[1].Tests[1].Severity)

	// Test column 2; test 3
	require.Equal(t, "unique", file.Models[0].Columns[1].Tests[2].Name)
	assert.Empty(t, file.Models[0].Columns[1].Tests[2].Arguments)
	assert.Empty(t, file.Models[0].Columns[1].Tests[2].Tags)
	assert.Equal(t, "warn", file.Models[0].Columns[1].Tests[2].Severity)

	// Test output back
	bytes, err := yaml.Marshal(file)
	require.NoError(t, err, "Unable to marshal properties file back to YAML")
	assert.Equal(t, yml, string(bytes), "Output file didn't match")
}
