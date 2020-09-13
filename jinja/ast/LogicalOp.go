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
	lhs, err := op.lhs.Execute(ec)
	if err != nil {
		return nil, err
	}
	if lhs == nil {
		return nil, ec.NilResultFor(op.lhs)
	}

	rhs, err := op.rhs.Execute(ec)
	if err != nil {
		return nil, err
	}
	if rhs == nil {
		return nil, ec.NilResultFor(op.rhs)
	}

	result := false

	switch op.op {
	case lexer.IsEqualsToken:
		result = lhs.Equals(rhs)

	case lexer.NotEqualsToken:
		result = !lhs.Equals(rhs)

	case lexer.LessThanToken, lexer.LessThanEqualsToken, lexer.GreaterThanToken, lexer.GreaterThanEqualsToken:
		lhsNum, err := lhs.AsNumberValue()
		if err != nil {
			return nil, ec.ErrorAt(op.lhs, fmt.Sprintf("%s", err))
		}

		rhsNum, err := rhs.AsNumberValue()
		if err != nil {
			return nil, ec.ErrorAt(op.rhs, fmt.Sprintf("%s", err))
		}

		switch op.op {
		case lexer.LessThanToken:
			result = lhsNum < rhsNum
		case lexer.LessThanEqualsToken:
			result = lhsNum <= rhsNum
		case lexer.GreaterThanToken:
			result = lhsNum > rhsNum
		case lexer.GreaterThanEqualsToken:
			result = lhsNum >= rhsNum
		}

	default:
		return nil, ec.ErrorAt(op, fmt.Sprintf("Unable to process logical operator `%s`", op.op))
	}

	return compilerInterface.NewBoolean(result), nil
}

func (op *LogicalOp) String() string {
	return fmt.Sprintf("(%s %s %s)", op.lhs, op.op, op.rhs)
}
