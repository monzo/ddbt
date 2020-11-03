package ast

import (
	"fmt"
	"strings"

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

	switch haystack.Type() {
	case compilerInterface.StringVal:
		// substring check
		needleStr := needle.AsStringValue()

		return compilerInterface.NewBoolean(strings.Contains(haystack.StringValue, needleStr)), nil

	case compilerInterface.ListVal:
		// value check
		for _, item := range haystack.ListValue {
			if item.Equals(needle) {
				return compilerInterface.NewBoolean(true), nil
			}
		}

		return compilerInterface.NewBoolean(false), nil

	case compilerInterface.MapVal:
		// key check
		needleStr := needle.AsStringValue()

		_, found := haystack.MapValue[needleStr]
		return compilerInterface.NewBoolean(found), nil

	default:
		return nil, ec.ErrorAt(in, fmt.Sprintf("Unable to perform the `in` operation on a %s", haystack.Type()))
	}
}

func (in *InOperator) String() string {
	return fmt.Sprintf("%s in %s", in.needle, in.haystack)
}
