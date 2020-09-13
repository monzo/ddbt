package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ddbt/compiler"
	"ddbt/compilerInterface"
	"ddbt/fs"
)

var testVariables = map[string]*compilerInterface.Value{
	"table_name":       {StringValue: "BLAH"},
	"number_value":     {NumberValue: 1},
	"str_number_value": {StringValue: "2"},
	"map_object": {
		MapValue: map[string]*compilerInterface.Value{
			"string": {StringValue: "test"},
			"nested": {
				MapValue: map[string]*compilerInterface.Value{
					"number": {NumberValue: 3},
					"string": {StringValue: "FROM"},
				},
			},
			"key": {StringValue: "42"},
		},
	},
	"list_object": {
		ListValue: []*compilerInterface.Value{
			{StringValue: "first option is string"},
			{StringValue: "second option a string too!"},
			{StringValue: "third"},
			{MapValue: map[string]*compilerInterface.Value{
				"blah": {ListValue: []*compilerInterface.Value{
					{StringValue: "thingy"},
				}},
			}},
			{ListValue: []*compilerInterface.Value{
				{StringValue: "nested list test"},
				{NumberValue: 3},
			}},
		},
	},
}

func TestNoJinjaTemplating(t *testing.T) {
	const raw = "SELECT * FROM BLAH"
	assertCompileOutput(t, raw, raw)
}

func TestCommentBlocks(t *testing.T) {
	const raw = "SELECT * FROM BLAH"
	assertCompileOutput(t, raw, "{# Test Comment #}"+raw)
	assertCompileOutput(t, raw, "{# Test Comment #}\n\t"+raw)
	assertCompileOutput(t, raw, raw+"{# Test Comment #}")
	assertCompileOutput(t, raw, raw+"\n\t{# Test Comment #}")
	assertCompileOutput(t, raw, "SELECT {# test comment#}* FROM BLAH")
	assertCompileOutput(t, raw, "SELECT {# test \n\n\ttest\n\ncomment#}* FROM BLAH")
}

func TestBasicVariables(t *testing.T) {
	assertCompileOutput(t, "BLAH", "{{ table_name }}")
	assertCompileOutput(t, "1", "{{ number_value }}")
	assertCompileOutput(t, "2", "{{ str_number_value }}")

	const raw = "SELECT * FROM BLAH"
	assertCompileOutput(t, raw, "SELECT * FROM {{ table_name }}")
	assertCompileOutput(t, raw, "SELECT * FROM {{table_name}}")
	assertCompileOutput(t, raw, "SELECT * FROM {{ table_name}}")
	assertCompileOutput(t, raw, "SELECT * FROM {{table_name }}")
}

func TestListVariables(t *testing.T) {
	assertCompileOutput(t, "first option is string", "{{ list_object[0] }}")
	assertCompileOutput(t, "second option a string too!", "{{ list_object[1] }}")
	assertCompileOutput(t, "third", "{{ list_object[2] }}")
}

func TestMapVariables(t *testing.T) {
	assertCompileOutput(t, "test", "{{ map_object['string'] }}")
	assertCompileOutput(t, "42", "{{ map_object['key'] }}")
	assertCompileOutput(t, "test", "{{ map_object.string }}")
	assertCompileOutput(t, "42", "{{ map_object.key }}")
}

func TestComplexVariableCombination(t *testing.T) {
	assertCompileOutput(t, "3", "{{ map_object.nested.number }}")
	assertCompileOutput(t, "thingy", "{{ list_object[3].blah[0] }}")
	assertCompileOutput(t, "thingy", "{{ list_object[map_object.nested.number].blah[0] }}")
	assertCompileOutput(t, "thingy", "{{ list_object[map_object['nested']['number']].blah[0] }}")
	assertCompileOutput(t, "thingy", "{{ list_object[list_object[4][1]].blah[0] }}")
}

func compileFromRaw(t *testing.T, raw string) string {
	fileSystem, err := fs.InMemoryFileSystem(
		map[string]string{
			"models/target_model.sql": raw,
		},
	)
	require.NoError(t, err, "Unable to construct in memory file system")

	for _, file := range fileSystem.AllFiles() {
		require.NoError(t, parseFile(file), "Unable to parse %s %s", file.Type, file.Name)
	}

	file := fileSystem.Model("target_model")
	require.NotNil(t, file, "Unable to extract the target_model from the In memory file system")
	require.NotNil(t, file.SyntaxTree, "target_model syntax tree is empty!")

	// Create the execution context
	ec := compiler.NewExecutionContext()
	for key, value := range testVariables {
		ec.SetVariable(key, value)
	}

	finalAST, err := file.SyntaxTree.Execute(ec)
	require.NoError(t, err)
	require.NotNil(t, finalAST, "Output AST is nil")
	require.Equal(t, compilerInterface.StringVal, finalAST.Type())

	return finalAST.StringValue
}

func assertCompileOutput(t *testing.T, expected, input string) {
	assert.Equal(
		t,
		expected,
		compileFromRaw(t, input),
		"Unexpected output from %s",
		input,
	)
}
