package ast

import (
	"ddbt/compilerInterface"
	"ddbt/jinja/lexer"
)

type BoolValue struct {
	position lexer.Position
	value    bool
}

var _ AST = &BoolValue{}

func NewBoolValue(token *lexer.Token) *BoolValue {
	return &BoolValue{
		position: token.Start,
		value:    token.Type == lexer.TrueToken,
	}
}

func (b *BoolValue) Position() lexer.Position {
	return b.position
}

func (b *BoolValue) Execute(ec compilerInterface.ExecutionContext) (*compilerInterface.Value, error) {
	return compilerInterface.NewBoolean(b.value), nil
}

func (b *BoolValue) String() string {
	if b.value {
		return "TRUE"
	} else {
		return "FALSE"
	}
}
