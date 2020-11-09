package ast

import (
	"fmt"

	"ddbt/compilerInterface"
	"ddbt/jinja/lexer"
)

type NotOperator struct {
	token        *lexer.Token
	subCondition AST
}

var _ AST = &NotOperator{}

func NewNotOperator(token *lexer.Token, subCondition AST) *NotOperator {
	return &NotOperator{
		token:        token,
		subCondition: subCondition,
	}
}

func (n *NotOperator) Position() lexer.Position {
	return n.token.Start
}

func (n *NotOperator) Execute(ec compilerInterface.ExecutionContext) (*compilerInterface.Value, error) {
	result, err := n.subCondition.Execute(ec)
	if err != nil {
		return nil, err
	}

	return compilerInterface.NewBoolean(!result.TruthyValue()), nil
}

func (n *NotOperator) String() string {
	return fmt.Sprintf("not %s", n.subCondition.String())
}

func (n *NotOperator) ApplyOperatorPrecedenceRules() AST {
	if and, ok := n.subCondition.(*AndCondition); ok {
		and.a = NewNotOperator(n.token, and.a).ApplyOperatorPrecedenceRules()

		return and
	}

	if or, ok := n.subCondition.(*OrCondition); ok {
		or.a = NewNotOperator(n.token, or.a).ApplyOperatorPrecedenceRules()

		return or
	}

	return n
}
