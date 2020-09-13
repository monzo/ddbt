package ast

import (
	"fmt"

	"ddbt/compilerInterface"
	"ddbt/jinja/lexer"
)

type UniaryMathsOp struct {
	position lexer.Position
	op       lexer.TokenType
	value    AST
}

var _ AST = &UniaryMathsOp{}

func NewUniaryMathsOp(token *lexer.Token, value AST) *UniaryMathsOp {
	return &UniaryMathsOp{
		position: token.Start,
		op:       token.Type,
		value:    value,
	}
}

func (op *UniaryMathsOp) Position() lexer.Position {
	return op.position
}

func (op *UniaryMathsOp) Execute(ec compilerInterface.ExecutionContext) (*compilerInterface.Value, error) {
	return nil, nil
}

func (op *UniaryMathsOp) String() string {
	return fmt.Sprintf("%s%s", op.op, op.value)
}
