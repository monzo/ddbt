package properties

import (
	"gopkg.in/yaml.v2"
)

// Represents what can be held within a DBT properties file
type File struct {
	Version   int    `yaml:"version"`             // What version of the schema we're on (always 2)
	Models    Models `yaml:"models,omitempty"`    // List of the model schemas defined in this file
	Macros    Macros `yaml:"macros,omitempty"`    // List of the macro schemas defined in this file
	Seeds     Models `yaml:"seeds,omitempty"`     // List of the seed schemas defined in this file (same structure as a model)
	Snapshots Models `yaml:"snapshots,omitempty"` // List of the snapshot schemas defined in this file (same structure as a model)
}

// The docs struct defines if a schema shows up on the docs server
type Docs struct {
	Show *bool `yaml:"show,omitempty"` // If not set, we default to true (but need to track it's not set for when we write YAML back out)
}

type Models []Model

// A model/seed/snapshot schema
type Model struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description,omitempty"`
	Docs        Docs     `yaml:"docs,omitempty"`
	Meta        MetaData `yaml:"meta,omitempty"`
	Tests       Tests    `yaml:"tests,omitempty"`   // Model level tests
	Columns     Columns  `yaml:"columns,omitempty"` // Columns
}

type Columns []Column

// Represents a single column within a Model schema
type Column struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description,omitempty"`
	Meta        MetaData `yaml:"meta,omitempty"`
	Quote       bool     `yaml:"quote,omitempty"`
	Tests       Tests    `yaml:"tests,omitempty"`
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
