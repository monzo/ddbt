package ast

import (
	"fmt"

	"ddbt/compilerInterface"
	"ddbt/jinja/lexer"
)

type OrCondition struct {
	a AST
	b AST
}

var _ AST = &OrCondition{}

func NewOrCondition(a, b AST) *OrCondition {
	return &OrCondition{
		a: a,
		b: b,
	}
}

func (o *OrCondition) Position() lexer.Position {
	return o.a.Position()
}

func (o *OrCondition) Execute(ec compilerInterface.ExecutionContext) (*compilerInterface.Value, error) {
	return nil, nil
}

func (o *OrCondition) String() string {
	return fmt.Sprintf("(%s or %s)", o.a.String(), o.b.String())
}
