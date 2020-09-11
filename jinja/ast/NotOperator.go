package ast

import (
	"fmt"

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

func (n *NotOperator) Execute(_ *ExecutionContext) AST {
	return nil
}

func (n *NotOperator) String() string {
	return fmt.Sprintf("not %s", n.subCondition.String())
}
