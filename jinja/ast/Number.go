package ast

import (
	"fmt"

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

func (n *Number) Execute(_ *ExecutionContext) AST {
	return nil
}

func (n *Number) String() string {
	return fmt.Sprintf("%s", n.value)
}
