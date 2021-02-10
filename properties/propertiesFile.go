package properties

import (
	"fmt"

	"gopkg.in/yaml.v2"
)

const FileVersion = 2

// Represents what can be held within a DBT properties file
type File struct {
	Version   int    `yaml:"version"`             // What version of the schema we're on (always 2)
	Models    Models `yaml:"models,omitempty"`    // List of the model schemas defined in this file
	Macros    Macros `yaml:"macros,omitempty"`    // List of the macro schemas defined in this file
	Seeds     Models `yaml:"seeds,omitempty"`     // List of the seed schemas defined in this file (same structure as a model)
	Snapshots Models `yaml:"snapshots,omitempty"` // List of the snapshot schemas defined in this file (same structure as a model)
}

// Unmarshals the file
func (f *File) Unmarshal(bytes []byte) error {
	return yaml.Unmarshal(bytes, f)
}

// Lists all the defined tests in this file
// returns a map with the test name as the key and test file jinja as the value
func (f *File) DefinedTests() (map[string]string, error) {
	tests := make(map[string]string)

	for _, model := range f.Models {
		if err := model.definedTests(tests); err != nil {
			return nil, err
		}
	}

	for _, model := range f.Seeds {
		if err := model.definedTests(tests); err != nil {
			return nil, err
		}
	}

	for _, model := range f.Snapshots {
		if err := model.definedTests(tests); err != nil {
			return nil, err
		}
	}

	return tests, nil
}

// The docs struct defines if a schema shows up on the docs server
type Docs struct {
	Show *bool `yaml:"show,omitempty"` // If not set, we default to true (but need to track it's not set for when we write YAML back out)
}

type Models []*Model

// A model/seed/snapshot schema
type Model struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Docs        Docs     `yaml:"docs,omitempty"`
	Meta        MetaData `yaml:"meta,omitempty"`
	Tests       Tests    `yaml:"tests,omitempty"`   // Model level tests
	Columns     Columns  `yaml:"columns,omitempty"` // Columns
}

func (m *Model) definedTests(tests map[string]string) error {
	// Table level tests
	for index, tableTest := range m.Tests {
		testName := fmt.Sprintf("%s_%s_%d", tableTest.Name, m.Name, index)
		jinja, err := tableTest.toTestJinja(m.Name, "")
		if err != nil {
			return err
		}

		tests[testName] = jinja
	}

	// Column level tests
	for _, column := range m.Columns {
		for index, tableTest := range column.Tests {
			testName := fmt.Sprintf("%s_%s__%s_%d", tableTest.Name, m.Name, column.Name, index)
			jinja, err := tableTest.toTestJinja(m.Name, column.Name)
			if err != nil {
				return err
			}

			tests[testName] = jinja
		}
	}

	return nil
}

type Columns []Column

// Represents a single column within a Model schema
type Column struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Meta        MetaData `yaml:"meta,omitempty"`
	Quote       bool     `yaml:"quote,omitempty"`
	Tests       Tests    `yaml:"tests"`
	Tags        []string `yaml:"tags,omitempty,flow"`
}

type Macros []Macro

// A macro schema
type Macro struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description,omitempty"`
	Docs        Docs           `yaml:"docs,omitempty"`
	Arguments   MacroArguments `yaml:"arguments,omitempty"`
}

type MacroArguments []MacroArgument

// A single argument on a macro
type MacroArgument struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
}

// Metadata that we can store against various parts of the schema
type MetaData yaml.MapSlice
