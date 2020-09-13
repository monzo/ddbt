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

func (a *AndCondition) Execute(ec compilerInterface.ExecutionContext) (*compilerInterface.Value, error) {
	// Execute the LHS
	result, err := a.a.Execute(ec)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ec.NilResultFor(a.a)
	}

	// Short circuit
	if !result.TruthyValue() {
		return compilerInterface.NewBoolean(false), nil
	}

	// Execute the RHS
	result, err = a.b.Execute(ec)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ec.NilResultFor(a.b)
	}

	return compilerInterface.NewBoolean(result.TruthyValue()), nil
}

func (a *AndCondition) String() string {
	return fmt.Sprintf("(%s and %s)", a.a.String(), a.b.String())
}
