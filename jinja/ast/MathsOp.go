package ast

import (
	"fmt"

	"ddbt/jinja/lexer"
)

type MathsOp struct {
	position lexer.Position
	lhs      AST
	rhs      AST
	op       lexer.TokenType
}

var _ AST = &MathsOp{}

func NewMathsOp(token *lexer.Token, lhs, rhs AST) *MathsOp {
	return &MathsOp{
		position: token.Start,
		lhs:      lhs,
		rhs:      rhs,
		op:       token.Type,
	}
}

func (op *MathsOp) Position() lexer.Position {
	return op.position
}

func (op *MathsOp) Execute(_ *ExecutionContext) AST {
	return nil
}

func (op *MathsOp) String() string {
	return fmt.Sprintf("%s %s %s", op.lhs, op.op, op.rhs)
}
