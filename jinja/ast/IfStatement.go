package ast

import (
	"fmt"

	"ddbt/compilerInterface"
	"ddbt/jinja/lexer"
)

type IfStatement struct {
	condition AST
	body      *Body
	elseBody  *Body
	asElseIf  bool
}

var _ AST = &IfStatement{}

func NewIfStatement(token *lexer.Token, condition AST) *IfStatement {
	return &IfStatement{
		condition: condition,
		body:      NewBody(token),
		elseBody:  NewBody(token),
	}
}

func (is *IfStatement) Position() lexer.Position {
	return is.condition.Position()
}

func (is *IfStatement) Execute(ec compilerInterface.ExecutionContext) (compilerInterface.AST, error) {
	return nil, nil
}

func (is *IfStatement) String() string {
	if len(is.elseBody.parts) > 0 {
		return fmt.Sprintf("{%% if %s %%}%s{%% else %%}%s{%% endif %%}", is.condition.String(), is.body.String(), is.elseBody.String())
	} else {
		return fmt.Sprintf("{%% if %s %%}%s{%% endif %%}", is.condition.String(), is.body.String())
	}
}

func (is *IfStatement) AppendBody(node AST) {
	is.body.Append(node)
}

func (is *IfStatement) AppendElse(node AST) {
	is.elseBody.Append(node)
}

func (is *IfStatement) SetAsElseIf() {
	is.asElseIf = true
}

func (is *IfStatement) IsElseIf() bool {
	return is.asElseIf
}
