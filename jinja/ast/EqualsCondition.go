package ast

import (
	"fmt"

	"ddbt/jinja/lexer"
)

type EqualsCondition struct {
	a AST
	b AST
}

var _ AST = &EqualsCondition{}

func NewEqualsCondition(a, b AST) *EqualsCondition {
	return &EqualsCondition{
		a: a,
		b: b,
	}
}

func (e *EqualsCondition) Position() lexer.Position {
	return e.a.Position()
}

func (e *EqualsCondition) Execute(_ *ExecutionContext) AST {
	return nil
}

func (e *EqualsCondition) String() string {
	return fmt.Sprintf("(%s == %s)", e.a.String(), e.b.String())
}
