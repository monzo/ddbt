package ast

import (
	"fmt"

	"ddbt/compilerInterface"
	"ddbt/jinja/lexer"
)

type Number struct {
	position lexer.Position
	value    string
}

var _ AST = &Number{}

func NewNumber(token *lexer.Token) *Number {
	return &Number{
		position: token.Start,
		value:    token.Value,
	}
}

func (n *Number) Position() lexer.Position {
	return n.position
}

func (n *Number) Execute(ec compilerInterface.ExecutionContext) (compilerInterface.AST, error) {
	return newTextBlockAt(n.position, n.value), nil
}

func (n *Number) String() string {
	return fmt.Sprintf("%s", n.value)
}
