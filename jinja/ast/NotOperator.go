package ast

import (
	"fmt"

	"ddbt/compilerInterface"
	"ddbt/jinja/lexer"
)

type NotOperator struct {
	position     lexer.Position
	subCondition AST
}

var _ AST = &NotOperator{}

func NewNotOperator(token *lexer.Token, subCondition AST) *NotOperator {
	return &NotOperator{
		position:     token.Start,
		subCondition: subCondition,
	}
}

func (n *NotOperator) Position() lexer.Position {
	return n.position
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
