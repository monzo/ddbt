package ast

import (
	"fmt"

	"ddbt/compilerInterface"
	"ddbt/jinja/lexer"
)

type DoBlock struct {
	position lexer.Position
	run      AST
}

var _ AST = &DoBlock{}

func NewDoBlock(token *lexer.Token, run AST) *DoBlock {
	return &DoBlock{
		position: token.Start,
		run:      run,
	}
}

func (d *DoBlock) Position() lexer.Position {
	return d.position
}

func (d *DoBlock) Execute(ec compilerInterface.ExecutionContext) (*compilerInterface.Value, error) {
	return nil, nil
}

func (d *DoBlock) String() string {
	return fmt.Sprintf("{%% do %s %%}", d.run.String())
}
