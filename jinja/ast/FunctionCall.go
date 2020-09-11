package ast

import (
	"fmt"
	"strings"

	"ddbt/jinja/lexer"
)

type FunctionCall struct {
	position  lexer.Position
	name      string
	arguments []funcCallArg
}

type funcCallArg struct {
	name string
	arg  AST
}

var _ AST = &FunctionCall{}

func NewFunctionCall(token *lexer.Token) *FunctionCall {
	return &FunctionCall{
		position:  token.Start,
		name:      token.Value,
		arguments: make([]funcCallArg, 0),
	}
}

func (fc *FunctionCall) Position() lexer.Position {
	return fc.position
}

func (fc *FunctionCall) Execute(_ *ExecutionContext) AST {
	return nil
}

func (fc *FunctionCall) String() string {
	var builder strings.Builder

	for i, arg := range fc.arguments {
		if i > 0 {
			builder.WriteString(", ")
		}

		if arg.name != "" {
			builder.WriteString(arg.name)
			builder.WriteRune('=')
		}

		builder.WriteString(arg.arg.String())
	}

	return fmt.Sprintf("{{ %s(%s) }}", fc.name, builder.String())
}

func (fc *FunctionCall) AddArgument(argName string, node AST) {
	fc.arguments = append(fc.arguments, funcCallArg{argName, node})
}
