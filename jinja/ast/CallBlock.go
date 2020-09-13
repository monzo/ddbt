package ast

import (
	"fmt"

	"ddbt/compilerInterface"
	"ddbt/jinja/lexer"
)

type CallBlock struct {
	position lexer.Position
	fc       *FunctionCall
	body     *Body
}

var _ AST = &CallBlock{}

func NewCallBlock(token *lexer.Token, fc *FunctionCall) *CallBlock {
	return &CallBlock{
		position: token.Start,
		fc:       fc,
		body:     NewBody(token),
	}
}

func (cb *CallBlock) Position() lexer.Position {
	return cb.position
}

func (cb *CallBlock) Execute(ec compilerInterface.ExecutionContext) (*compilerInterface.Value, error) {
	return nil, nil
}

func (cb *CallBlock) String() string {
	return fmt.Sprintf("{%% call %s %%}%s\n{%% endcall %%}", cb.fc.String(), cb.body.String())
}

func (cb *CallBlock) AppendBody(node AST) {
	cb.body.Append(node)
}
