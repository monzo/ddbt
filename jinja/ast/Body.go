package ast

import (
	"fmt"
	"reflect"
	"strings"

	"ddbt/compilerInterface"
	"ddbt/jinja/lexer"
)

// A block which represents a simple
type Body struct {
	position lexer.Position
	parts    []AST
}

var _ AST = &Body{}

func NewBody(token *lexer.Token) *Body {
	return &Body{
		position: token.Start,
		parts:    make([]AST, 0),
	}
}

func (b *Body) Position() lexer.Position {
	return b.position
}

func (b *Body) Execute(ec compilerInterface.ExecutionContext) (*compilerInterface.Value, error) {
	// A body should compile down to only text blocks
	var builder strings.Builder

	for _, part := range b.parts {
		result, err := part.Execute(ec)
		if err != nil {
			return nil, err
		}

		if err := writeValue(ec, part, &builder, result); err != nil {
			return nil, err
		}

		if result.ValueType == compilerInterface.ReturnVal {
			return result, nil
		}
	}

	value := compilerInterface.NewString(builder.String())

	return value, nil
}

func (b *Body) String() string {
	var builder strings.Builder

	for _, part := range b.parts {
		builder.WriteString(part.String())
	}

	return builder.String()
}

// Append a node to the body
func (b *Body) Append(node AST) {
	b.parts = append(b.parts, node)
}

func writeValue(ec compilerInterface.ExecutionContext, part compilerInterface.AST, builder *strings.Builder, value *compilerInterface.Value) error {
	if value == nil {
		return ec.NilResultFor(part)
	}

	t := value.Type()
	switch t {
	case compilerInterface.StringVal, compilerInterface.NumberVal, compilerInterface.BooleanValue, compilerInterface.ReturnVal:
		builder.WriteString(value.AsStringValue())

	case compilerInterface.ListVal:
		builder.WriteRune('[')
		for i, item := range value.ListValue {
			if i > 0 {
				builder.WriteString(", ")
			}

			if err := writeValue(ec, part, builder, item); err != nil {
				return err
			}
		}
		builder.WriteRune(']')

	case compilerInterface.MapVal:
		builder.WriteRune('{')
		i := 0
		for key, item := range value.MapValue {
			if i > 0 {
				builder.WriteString(", ")
			}

			builder.WriteRune('"')
			builder.WriteString(key)
			builder.WriteString("\": ")

			if err := writeValue(ec, part, builder, item); err != nil {
				return err
			}
			i++
		}
		builder.WriteRune('}')

	case compilerInterface.Undefined, compilerInterface.NullVal:
	// no-op as we can consume these without effect

	default:
		return ec.ErrorAt(
			part,
			fmt.Sprintf(
				"A %v returned a %s which can not be combined into a body",
				reflect.TypeOf(part),
				t,
			),
		)
	}

	return nil
}
