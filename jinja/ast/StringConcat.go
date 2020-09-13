package ast

import (
	"fmt"

	"ddbt/compilerInterface"
	"ddbt/jinja/lexer"
)

type StringConcat struct {
	position lexer.Position
	lhs      AST
	rhs      AST
}

var _ AST = &StringConcat{}

func NewStringConcat(token *lexer.Token, lhs, rhs AST) *StringConcat {
	return &StringConcat{
		position: token.Start,
		lhs:      lhs,
		rhs:      rhs,
	}
}

func (sc *StringConcat) Position() lexer.Position {
	return sc.position
}

func (sc *StringConcat) Execute(ec compilerInterface.ExecutionContext) (*compilerInterface.Value, error) {
	lhs, err := sc.lhs.Execute(ec)
	if err != nil {
		return nil, err
	}
	if lhs == nil {
		return nil, ec.NilResultFor(sc.lhs)
	}

	rhs, err := sc.rhs.Execute(ec)
	if err != nil {
		return nil, err
	}
	if rhs == nil {
		return nil, ec.NilResultFor(sc.rhs)
	}

	return compilerInterface.NewString(lhs.AsStringValue() + rhs.AsStringValue()), nil
}

func (sc *StringConcat) String() string {
	return fmt.Sprintf("%s ~ %s", sc.lhs.String(), sc.rhs.String())
}
