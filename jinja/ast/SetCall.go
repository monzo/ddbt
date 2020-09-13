package ast

import (
	"fmt"

	"ddbt/compilerInterface"
	"ddbt/jinja/lexer"
)

type SetCall struct {
	position      lexer.Position
	variableToSet string
	condition     AST
}

var _ AST = &SetCall{}

func NewSetCall(ident *lexer.Token, condition AST) *SetCall {
	return &SetCall{
		position:      ident.Start,
		variableToSet: ident.Value,
		condition:     condition,
	}
}

func (sc *SetCall) Position() lexer.Position {
	return sc.position
}

func (sc *SetCall) Execute(ec compilerInterface.ExecutionContext) (compilerInterface.AST, error) {
	return nil, nil
}

func (sc *SetCall) String() string {
	return fmt.Sprintf("{%% set %s = %s %%}", sc.variableToSet, sc.condition.String())
}
