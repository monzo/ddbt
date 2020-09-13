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

func (n *NullValue) Execute(_ compilerInterface.ExecutionContext) (*compilerInterface.Value, error) {
	return &compilerInterface.Value{ValueType: compilerInterface.NullVal, IsNull: true}, nil
}

func (n *NullValue) String() string {
	return "null"
}
