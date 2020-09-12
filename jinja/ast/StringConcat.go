package ast

import (
	"fmt"

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

func (sc *StringConcat) Execute(_ *ExecutionContext) AST {
	return nil
}

func (sc *StringConcat) String() string {
	return fmt.Sprintf("%s ~ %s", sc.lhs.String(), sc.rhs.String())
}
