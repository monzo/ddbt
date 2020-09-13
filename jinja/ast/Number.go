package ast

import (
	"fmt"

	"ddbt/compilerInterface"
	"ddbt/jinja/lexer"
)

type Number struct {
	position lexer.Position
	number   float64
}

var _ AST = &Number{}

func NewNumber(token *lexer.Token, number float64) *Number {
	return &Number{
		position: token.Start,
		number:   number,
	}
}

func (n *Number) Position() lexer.Position {
	return n.position
}

func (n *Number) Execute(_ compilerInterface.ExecutionContext) (*compilerInterface.Value, error) {
	return compilerInterface.NewNumber(n.number), nil
}

func (n *Number) String() string {
	return fmt.Sprintf("%g", n.number)
}
