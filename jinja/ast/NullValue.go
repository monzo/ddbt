package ast

import (
	"ddbt/compilerInterface"
	"ddbt/jinja/lexer"
)

type NullValue struct {
	position lexer.Position
}

var _ AST = &NullValue{}

func NewNullValue(token *lexer.Token) *NullValue {
	return &NullValue{
		position: token.Start,
	}
}

func (n *NullValue) Position() lexer.Position {
	return n.position
}

func (n *NullValue) Execute(_ compilerInterface.ExecutionContext) (compilerInterface.AST, error) {
	return newTextBlockAt(n.position, ""), nil
}

func (n *NullValue) String() string {
	return "null"
}
