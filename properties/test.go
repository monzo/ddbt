package properties

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"gopkg.in/yaml.v2"
)

type Tests []*Test

// Represents a test we should perform against a Model or Column
//
// Due to how DBT encodes these within YAML we need to write a custom set
// of marshallers to ensure we can read/write the YAML files correctly
//
// Inside the tests slice the YAML could look like this when there are no other parameters;
//
// - test_name
//
// or like this, if we need to provide other params;
//
// - test_name:
//     arg1: something
//     tags: ['a', 'b', 'c']
type Test struct {
	Name      string // the name of the inbuilt test we want to run such as "not_null" or "unique"
	Severity  string // "warn" or "error" are the only allowed values
	Tags      []string
	Arguments TestArguments
}

// The arguments which a test requires
// Note; we use a slice here to preserve ordering if we are mutating an existing schema file
// on the filesystem
type TestArguments []TestArgument
type TestArgument struct {
	Name  string
	Value interface{}
}

func (o *Test) UnmarshalYAML(unmarshal func(v interface{}) error) error {
	var m map[string]yaml.MapSlice

	// Handle the case where we just have a test name
	if err := unmarshal(&m); err != nil {
		if err2 := unmarshal(&o.Name); err2 == nil {
			return nil
		}

		return err
	}

	// Otherwise we expect a map with a single key - the test name
	if len(m) != 1 {
		return errors.New(fmt.Sprintf("expected 1 key for test, got %d", len(m)))
	}

	// Now read the arguments out in the order they are present
	for testName, properties := range m {
		o.Name = testName
		arguments := make(TestArguments, 0)

		for _, property := range properties {
			// pull out the severity or tags key to the top level test object
			switch property.Key {
			case "severity":
				switch v := property.Value.(type) {
				case string:
					if v != "warn" && v != "error" {
						return errors.New(fmt.Sprintf("severity expected to be a `warn` or `error`, got %v", v))
					}

					o.Severity = v
				default:
					return errors.New(fmt.Sprintf("severity expected to be a `warn` or `error`, got %v", reflect.TypeOf(v)))
				}

			case "tags":
				switch v := property.Value.(type) {
				case []interface{}:
					tags := make([]string, 0, len(v))

					for _, tagI := range v {
						if tag, ok := tagI.(string); ok {
							tags = append(tags, tag)
						} else {
							return errors.New(fmt.Sprintf("expected tag value to be a string, got %v", reflect.TypeOf(tagI)))
						}
					}

					o.Tags = tags

				default:
					return errors.New(fmt.Sprintf("tags expected to be an array, got %v", reflect.TypeOf(v)))
				}

			default:
				// otherwise transpose the test arguments to our arguments struct
				str, ok := property.Key.(string)
				if !ok {
					return errors.New(fmt.Sprintf("unable to convert property key to string: %v", property.Key))
				}

				arguments = append(arguments, TestArgument{
					Name:  str,
					Value: property.Value,
				})
			}
		}

		o.Arguments = arguments
	}

	return nil
}

func (o *Test) MarshalYAML() (interface{}, error) {
	if len(o.Arguments) == 0 && len(o.Tags) == 0 && o.Severity == "" {
		return o.Name, nil
	}

	args := make(yaml.MapSlice, 0)

	// Write the arguments back out to a MapSlice (to preserve ordering)
	for _, arg := range o.Arguments {
		args = append(args, yaml.MapItem{Key: arg.Name, Value: arg.Value})
	}

	// Append our tags
	if len(o.Tags) > 0 {
		args = append(args, yaml.MapItem{Key: "tags", Value: o.Tags})
	}

	// Append the Severity
	if o.Severity != "" {
		args = append(args, yaml.MapItem{Key: "severity", Value: o.Severity})
	}

	return yaml.MapSlice{
		{Key: o.Name, Value: args},
	}, nil
}

// Converts this test to a Jinja comptible format
func (o *Test) toTestJinja(tableName, columnName string) (string, error) {
	var builder strings.Builder

	builder.WriteString("{{ test_")
	builder.WriteString(o.Name)

	builder.WriteString("( model=ref('")
	builder.WriteString(tableName)
	builder.WriteString("')")

	if columnName != "" {
		builder.WriteString(", column_name='")
		builder.WriteString(columnName)
		builder.WriteString("'")
	}

	for _, arg := range o.Arguments {
		jsonValue, err := json.Marshal(arg.Value)
		if err != nil {
			return "", errors.New(
				fmt.Sprintf(
					"Unable to convert parameter for test %s on column %s of table %s: %s",
					o.Name,
					columnName,
					tableName,
					err.Error(),
				),
			)
		}

		builder.WriteString(", ")
		builder.WriteString(arg.Name)
		builder.WriteRune('=')
		builder.Write(jsonValue)
	}

	builder.WriteString(") }}")

	return builder.String(), nil
}
