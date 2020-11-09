package ast

import (
	"fmt"

	"ddbt/compilerInterface"
	"ddbt/jinja/lexer"
)

type InOperator struct {
	position lexer.Position
	needle   AST
	haystack AST
}

var _ AST = &InOperator{}

func NewInOperator(token *lexer.Token, needle, haystack AST) *InOperator {
	return &InOperator{
		position: token.Start,
		needle:   needle,
		haystack: haystack,
	}
}

func (in *InOperator) Position() lexer.Position {
	return in.position
}

func (in *InOperator) Execute(ec compilerInterface.ExecutionContext) (*compilerInterface.Value, error) {
	needle, err := in.needle.Execute(ec)
	if err != nil {
		return nil, err
	}

	haystack, err := in.haystack.Execute(ec)
	if err != nil {
		return nil, err
	}

	// If these are functions results, strip the result wrapper
	needle = needle.Unwrap()
	haystack = haystack.Unwrap()

	result, err := BuiltInTests["in"](needle, haystack)
	if err != nil {
		return nil, ec.ErrorAt(in.haystack, err.Error())
	}

	return compilerInterface.NewBoolean(result), nil
}

func (in *InOperator) String() string {
	return fmt.Sprintf("%s in %s", in.needle, in.haystack)
}
