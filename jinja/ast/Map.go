package ast

import "ddbt/jinja/lexer"

type Map struct {
	position lexer.Position
	data     map[string]*lexer.Token
}

var _ AST = &Map{}

func NewMap(token *lexer.Token) *Map {
	return &Map{
		position: token.Start,
		data:     make(map[string]*lexer.Token),
	}
}

func (m *Map) Position() lexer.Position {
	return m.position
}

func (m *Map) Execute(_ *ExecutionContext) AST {
	return nil
}

func (m *Map) String() string {
	return ""
}

func (m *Map) Put(key *lexer.Token, value *lexer.Token) {
	m.data[key.Value] = value
}
