package ast

import (
	"fmt"

	"ddbt/compilerInterface"
	"ddbt/jinja/lexer"
)

type LogicalOp struct {
	position lexer.Position
	op       lexer.TokenType
	lhs      AST
	rhs      AST
}

var _ AST = &LogicalOp{}

func NewLogicalOp(token *lexer.Token, lhs, rhs AST) *LogicalOp {
	return &LogicalOp{
		position: token.Start,
		op:       token.Type,
		lhs:      lhs,
		rhs:      rhs,
	}
}

func (op *LogicalOp) Position() lexer.Position {
	return op.position
}

func (op *LogicalOp) Execute(ec compilerInterface.ExecutionContext) (*compilerInterface.Value, error) {
	return nil, nil
}

func (op *LogicalOp) String() string {
	return fmt.Sprintf("(%s %s %s)", op.lhs, op.op, op.rhs)
}
