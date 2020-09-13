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
	value, err := op.value.Execute(ec)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, ec.NilResultFor(op.value)
	}
	valueNum, err := value.AsNumberValue()
	if err != nil {
		return nil, ec.ErrorAt(op.value, fmt.Sprintf("%s", err))
	}

	var result float64

	switch op.op {
	case lexer.PlusToken:
		result = +valueNum

	case lexer.MinusToken:
		result = -valueNum

	default:
		return nil, ec.ErrorAt(op, fmt.Sprintf("Unknown maths uniary operator `%s`", op.op))
	}

	return compilerInterface.NewNumber(result), nil
}

func (op *UniaryMathsOp) String() string {
	return fmt.Sprintf("%s%s", op.op, op.value)
}
