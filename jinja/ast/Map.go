package ast

import (
	"ddbt/compilerInterface"
	"ddbt/jinja/lexer"
)

type Map struct {
	position lexer.Position
	data     map[AST]AST
}

var _ AST = &Map{}

func NewMap(token *lexer.Token) *Map {
	return &Map{
		position: token.Start,
		data:     make(map[AST]AST),
	}
}

func (m *Map) Position() lexer.Position {
	return m.position
}

func (m *Map) Execute(ec compilerInterface.ExecutionContext) (*compilerInterface.Value, error) {
	resultMap := make(map[string]*compilerInterface.Value)

	for key, value := range m.data {
		key, err := key.Execute(ec)
		if err != nil {
			return nil, err
		}

		result, err := value.Execute(ec)
		if err != nil {
			return nil, err
		}
		if result == nil {
			return nil, ec.NilResultFor(value)
		}

		resultMap[key.AsStringValue()] = result
	}

	return &compilerInterface.Value{ValueType: compilerInterface.MapVal, MapValue: resultMap}, nil
}

func (m *Map) String() string {
	return ""
}

func (m *Map) Put(key AST, value AST) {
	m.data[key] = value
}
