package ast

import "ddbt/jinja/lexer"

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

func (n *NullValue) Execute(_ *ExecutionContext) AST {
	return nil
}

func (n *NullValue) String() string {
	return ""
}
