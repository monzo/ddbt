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
	// Execute the LHS
	result, err := o.a.Execute(ec)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ec.NilResultFor(o.a)
	}

	// Short circuit
	if result.TruthyValue() {
		return compilerInterface.NewBoolean(true), nil
	}

	// Execute the RHS
	result, err = o.b.Execute(ec)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ec.NilResultFor(o.b)
	}

	return compilerInterface.NewBoolean(result.TruthyValue()), nil
}

func (o *OrCondition) String() string {
	return fmt.Sprintf("(%s or %s)", o.a.String(), o.b.String())
}
