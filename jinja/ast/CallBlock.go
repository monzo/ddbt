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
	ec.PushState()
	defer ec.PopState()

	// Set it so the body AST can be executed using the caller function
	ec.SetVariable(
		"caller",
		compilerInterface.NewFunction(func(ec compilerInterface.ExecutionContext, caller compilerInterface.AST, args compilerInterface.Arguments) (*compilerInterface.Value, error) {
			return cb.body.Execute(ec)
		}),
	)

	// Execute the function call
	return cb.fc.Execute(ec)
}

func (cb *CallBlock) String() string {
	return fmt.Sprintf("{%% call %s %%}%s\n{%% endcall %%}", cb.fc.String(), cb.body.String())
}

func (cb *CallBlock) AppendBody(node AST) {
	cb.body.Append(node)
}
