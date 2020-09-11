package ast

import (
	"strings"

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

func (b *Body) Execute(_ *ExecutionContext) AST {
	return nil
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
