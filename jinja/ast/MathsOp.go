package ast

import (
	"fmt"
	"math"

	"ddbt/compilerInterface"
	"ddbt/jinja/lexer"
)

type MathsOp struct {
	token *lexer.Token
	lhs   AST
	rhs   AST
}

var _ AST = &MathsOp{}

func NewMathsOp(token *lexer.Token, lhs, rhs AST) *MathsOp {
	return &MathsOp{
		token: token,
		lhs:   lhs,
		rhs:   rhs,
	}
}

func (op *MathsOp) Position() lexer.Position {
	return op.token.Start
}

func (op *MathsOp) Execute(ec compilerInterface.ExecutionContext) (*compilerInterface.Value, error) {
	lhs, err := op.lhs.Execute(ec)
	if err != nil {
		return nil, err
	}
	if lhs == nil {
		return nil, ec.NilResultFor(op.lhs)
	}
	lhsNum, err := lhs.AsNumberValue()
	if err != nil {
		return nil, ec.ErrorAt(op.lhs, fmt.Sprintf("%s", err))
	}

	rhs, err := op.rhs.Execute(ec)
	if err != nil {
		return nil, err
	}
	if rhs == nil {
		return nil, ec.NilResultFor(op.rhs)
	}
	rhsNum, err := rhs.AsNumberValue()
	if err != nil {
		return nil, ec.ErrorAt(op.rhs, fmt.Sprintf("%s", err))
	}

	var result float64

	switch op.token.Type {
	case lexer.PlusToken:
		result = lhsNum + rhsNum

	case lexer.MinusToken:
		result = lhsNum - rhsNum

	case lexer.MultiplyToken:
		result = lhsNum * rhsNum

	case lexer.DivideToken:
		result = lhsNum / rhsNum

	case lexer.PowerToken:
		result = math.Pow(lhsNum, rhsNum)

	default:
		return nil, ec.ErrorAt(op, fmt.Sprintf("Unknown maths operator `%s`", op.token.Type))
	}

	return compilerInterface.NewNumber(result), nil
}

// The parse is a 1-token look ahead parser, so it will parse
// `2 + 3 * 4 + 5` into `+(2, *(3, +(4, 5)))` when due to operator
// precedence rules it should be `*(+(2, 3), +(4, 5))`.
//
// This function will rewrite the AST tree starting with this MathOps in a left to right manor
// this means this function should NEVER see a MathsOp as it's left hand side!
func (op *MathsOp) ApplyOperatorPrecedenceRules() *MathsOp {
	if _, ok := op.lhs.(*MathsOp); ok {
		// Invariant failure
		panic("got a MathsOp as the lhs during operator precedence reordering: " + op.String())
	}

	if rhs, ok := op.rhs.(*MathsOp); ok {
		if rhs.operatorPrecdence() <= op.operatorPrecdence() {
			// Example 1: 2 * 3 + 4
			// AST = *(2, +(3, 4))
			// Op = *, LHS = 2, RHS = +(3, 4)
			//
			// Rewrite as: (2 * 3) + 4
			// AST = +(*(2, 3), 4)
			// Op = +, LHS = *(2, 3), RHS = 4

			// Example 2: 20 / 4 * 5
			// AST = /(20, *(4, 5))
			// Op = /, LHS = 20, RHS = *(4, 5)
			//
			// Rewrite as: (20 / 4) * 5
			// Op  = *, LHS = /(20, 4), RHS(5)

			now := NewMathsOp(
				rhs.token,
				NewMathsOp(
					op.token,
					op.lhs,
					rhs.lhs,
				).ApplyOperatorPrecedenceRules(),
				rhs.rhs,
			)

			return now
		}
	}

	return op
}

func (op *MathsOp) operatorPrecdence() int {
	switch op.token.Type {
	case lexer.PowerToken:
		return 3
	case lexer.MultiplyToken, lexer.DivideToken:
		return 2
	case lexer.PlusToken, lexer.MinusToken:
		return 1
	default:
		panic("unknown op: " + op.token.Type)
	}
}

func (op *MathsOp) String() string {
	return fmt.Sprintf("(%s %s %s)", op.lhs, op.token.Type, op.rhs)
}
