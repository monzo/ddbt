package ast

import (
	"fmt"

	"ddbt/jinja/lexer"
)

type DefineCheck struct {
	position  lexer.Position
	condition AST
	checkType string
}

var _ AST = &DefineCheck{}

func NewDefineCheck(token *lexer.Token, condition AST, checkType string) *DefineCheck {
	return &DefineCheck{
		position:  token.Start,
		condition: condition,
		checkType: checkType,
	}
}

func (op *DefineCheck) Position() lexer.Position {
	return op.position
}

func (op *DefineCheck) Execute(_ *ExecutionContext) AST {
	return nil
}

func (op *DefineCheck) String() string {
	return fmt.Sprintf("%s is %s", op.condition.String(), op.checkType)
}
