package ast

import (
	"strings"

	"ddbt/jinja/lexer"
)

type Variable struct {
	position lexer.Position
	name     string

	subVariable *Variable

	argCall        []funcCallArg
	isNested       bool // Track if this variable is a sub variable (i.e. `b` in `a.b`)
	isMapLookup    bool // Track if this variable is a map lookup (i.e. `b` in `a["b"]`)
	isCalledAsFunc bool
}

var _ AST = &Variable{}

func NewVariable(token *lexer.Token) *Variable {
	return &Variable{
		position: token.Start,
		name:     token.Value,
		argCall:  make([]funcCallArg, 0),
	}
}

func (v *Variable) Position() lexer.Position {
	return v.position
}

func (v *Variable) Execute(_ *ExecutionContext) AST {
	return nil
}

func (v *Variable) String() string {
	var builder strings.Builder

	if !v.isNested {
		builder.WriteString("{{ ")
	} else if v.isMapLookup {
		builder.WriteString("[\"")
	} else {
		builder.WriteRune('.')
	}

	builder.WriteString(v.name)

	if v.subVariable != nil {
		builder.WriteString(v.subVariable.String())
	}

	if !v.isNested {
		builder.WriteString(" }}")
	} else if v.isMapLookup {
		builder.WriteString("\"]")
	}

	if v.isCalledAsFunc {
		builder.WriteRune('(')

		for i, arg := range v.argCall {
			if i > 0 {
				builder.WriteString(", ")
			}

			if arg.name != "" {
				builder.WriteString(arg.name)
				builder.WriteRune('=')
			}

			builder.WriteString(arg.arg.String())
		}

		builder.WriteRune(')')
	}

	return builder.String()
}

func (v *Variable) SetSub(subVariable *Variable) {
	v.subVariable = subVariable
	subVariable.isNested = true
}

func (v *Variable) SetMapLookup(subVariable *Variable) {
	v.subVariable = subVariable
	subVariable.isNested = true
	subVariable.isMapLookup = true
}

func (v *Variable) AddArgument(argName string, node AST) {
	v.argCall = append(v.argCall, funcCallArg{argName, node})
}

func (v *Variable) IsCalledAsFunc() {
	v.isCalledAsFunc = true
}
