package ast

import (
	"ddbt/compilerInterface"
	"ddbt/jinja/lexer"
)

type NoneValue struct {
	position lexer.Position
}

var _ AST = &NoneValue{}

func NewNoneValue(token *lexer.Token) *NoneValue {
	return &NoneValue{
		position: token.Start,
	}
}

func (n *NoneValue) Position() lexer.Position {
	return n.position
}

func (n *NoneValue) Execute(_ compilerInterface.ExecutionContext) (*compilerInterface.Value, error) {
	return compilerInterface.NewUndefined(), nil
}

func (n *NoneValue) String() string {
	return "None"
}
