package ast

import (
	"fmt"

	"ddbt/compilerInterface"
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

func (op *DefineCheck) Execute(ec compilerInterface.ExecutionContext) (*compilerInterface.Value, error) {
	value, err := op.condition.Execute(ec)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, ec.NilResultFor(op.condition)
	}

	result := false

	switch op.checkType {
	case "defined":
		result = !value.IsUndefined

	case "not defined":
		result = value.IsUndefined

	case "none":
		result = value.IsNull || value.IsUndefined

	case "not none":
		result = !value.IsNull && !value.IsUndefined

	default:
		return nil, ec.ErrorAt(op, fmt.Sprintf("Unknown define check type `%s`", op.checkType))

	}

	return compilerInterface.NewBoolean(result), nil
}

func (op *DefineCheck) String() string {
	return fmt.Sprintf("%s is %s", op.condition.String(), op.checkType)
}
