package ast

import (
	"fmt"

	"ddbt/compilerInterface"
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

func (a *AndCondition) Execute(ec compilerInterface.ExecutionContext) (compilerInterface.AST, error) {
	return nil, nil
}

func (a *AndCondition) String() string {
	return fmt.Sprintf("(%s and %s)", a.a.String(), a.b.String())
}
