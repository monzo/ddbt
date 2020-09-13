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

		if result == nil {
			return nil, ec.ErrorAt(
				b,
				fmt.Sprintf(
					"A %v did not return a value",
					reflect.TypeOf(part),
				),
			)
		}

		t := result.Type()
		switch t {
		case compilerInterface.StringVal:
			builder.WriteString(result.StringValue)

		case compilerInterface.NumberVal:
			builder.WriteString(fmt.Sprintf("%.f", result.NumberValue))

		case compilerInterface.Undefined, compilerInterface.NullVal:
		// no-op as we can consume these without effect

		default:
			return nil, ec.ErrorAt(
				part,
				fmt.Sprintf(
					"A %v returned a %s which can not be combined into a body",
					reflect.TypeOf(part),
					t,
				),
			)
		}
	}

	return &compilerInterface.Value{StringValue: strings.TrimSpace(builder.String())}, nil
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
