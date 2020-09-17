package tests

import (
	"ddbt/compiler"
	"ddbt/compilerInterface"
	"ddbt/config"
	"ddbt/fs"
	"ddbt/jinja"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

var debugPrintAST = false

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
	config.GlobalCfg = &config.Config{
		Name: "Unit Test",
		Target: &config.Target{
			Name:      "unit_test",
			ProjectID: "unit_test_project",
			DataSet:   "unit_test_dataset",
			Location:  "US",
			Threads:   4,
		},
	}
	gc := compiler.NewGlobalContext(config.GlobalCfg, fileSystem)
	ec := compiler.NewExecutionContext(file, fileSystem, gc)
	ec.SetVariable("config", file.ConfigObject())
	for key, value := range testVariables {
		ec.SetVariable(key, value)
	}

	finalAST, err := file.SyntaxTree.Execute(ec)
	require.NoError(t, err)
	require.NotNil(t, finalAST, "Output AST is nil")
	//require.Equal(t, compilerInterface.StringVal, finalAST.Type())

	return finalAST.AsStringValue()
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

func parseFile(file *fs.File) error {
	syntaxTree, err := jinja.Parse(file)
	if err != nil {
		return err
	}

	if debugPrintAST {
		debugPrintAST = false
		fmt.Println(syntaxTree.String())
	}

	file.SyntaxTree = syntaxTree
	return nil
}
