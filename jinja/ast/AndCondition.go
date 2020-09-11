package ast

import (
	"fmt"

	"ddbt/jinja/lexer"
)

type AndCondition struct {
	a AST
	b AST
}

var _ AST = &AndCondition{}

func NewAndCondition(a, b AST) *AndCondition {
	return &AndCondition{
		a: a,
		b: b,
	}
}

func (a *AndCondition) Position() lexer.Position {
	return a.a.Position()
}

func (a *AndCondition) Execute(_ *ExecutionContext) AST {
	return nil
}

func (a *AndCondition) String() string {
	return fmt.Sprintf("(%s and %s)", a.a.String(), a.b.String())
}
