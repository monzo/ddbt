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

func (b *Body) Execute(ec compilerInterface.ExecutionContext) (compilerInterface.AST, error) {
	// A body should compile down to only text blocks
	var builder strings.Builder

	for _, part := range b.parts {
		newPart, err := part.Execute(ec)
		if err != nil {
			return nil, err
		}

		if _, ok := newPart.(*TextBlock); !ok {
			return nil, ec.ErrorAt(
				newPart,
				fmt.Sprintf(
					"part did not compile down to plain text; got %v from %v",
					reflect.TypeOf(newPart),
					reflect.TypeOf(part),
				),
			)
		}

		builder.WriteString(newPart.String())
	}

	return newTextBlockAt(b.position, strings.TrimSpace(builder.String())), nil
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
