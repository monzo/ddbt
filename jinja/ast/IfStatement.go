package ast

import (
	"fmt"

	"ddbt/jinja/lexer"
)

type IfStatement struct {
	condition AST
	body      *Body
}

var _ AST = &IfStatement{}

func NewIfStatement(token *lexer.Token, condition AST) *IfStatement {
	return &IfStatement{
		condition: condition,
		body:      NewBody(token),
	}
}

func (is *IfStatement) Position() lexer.Position {
	return is.condition.Position()
}

func (is *IfStatement) Execute(_ *ExecutionContext) AST {
	return nil
}

func (is *IfStatement) String() string {
	return fmt.Sprintf("{%% if %s %%}%s{%% endif %%}", is.condition.String(), is.body.String())
}

func (is *IfStatement) AppendBody(node AST) {
	is.body.Append(node)
}
