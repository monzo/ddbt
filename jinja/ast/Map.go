package ast

import (
	"ddbt/compilerInterface"
	"ddbt/jinja/lexer"
)

type Map struct {
	position lexer.Position
	data     map[string]AST
}

var _ AST = &Map{}

func NewMap(token *lexer.Token) *Map {
	return &Map{
		position: token.Start,
		data:     make(map[string]AST),
	}
}

func (m *Map) Position() lexer.Position {
	return m.position
}

func (m *Map) Execute(ec compilerInterface.ExecutionContext) (compilerInterface.AST, error) {
	return nil, nil
}

func (m *Map) String() string {
	return ""
}

func (m *Map) Put(key *lexer.Token, value AST) {
	m.data[key.Value] = value
}
