package ast

import (
	"strings"

	"ddbt/jinja/lexer"
)

type variableType = int

const (
	identVar variableType = iota
	indexVar
	mapLookupVar
	funcCallVar
)

type Variable struct {
	token       *lexer.Token
	varType     variableType
	subVariable *Variable

	argCall         []funcCallArg
	lookupKey       AST
	isTemplateBlock bool
}

var _ AST = &Variable{}

func NewVariable(token *lexer.Token) *Variable {
	return &Variable{
		token:   token,
		varType: identVar,
		argCall: make([]funcCallArg, 0),
	}
}

func (v *Variable) Position() lexer.Position {
	return v.token.Start
}

func (v *Variable) Execute(_ *ExecutionContext) AST {
	return nil
}

func (v *Variable) String() string {
	var builder strings.Builder

	if v.isTemplateBlock {
		builder.WriteString("{{ ")
	}

	switch v.varType {
	case identVar:
		builder.WriteString(v.token.Value)

	case indexVar:
		builder.WriteString(v.subVariable.String())
		builder.WriteRune('.')
		builder.WriteString(v.token.Value)

	case mapLookupVar:
		builder.WriteString(v.subVariable.String())
		builder.WriteRune('[')
		builder.WriteString(v.lookupKey.String())
		builder.WriteRune(']')

	case funcCallVar:
		builder.WriteString(v.subVariable.String())
		builder.WriteRune('(')

		for i, arg := range v.argCall {
			if i > 0 {
				builder.WriteString(", ")
			}

			if arg.name != "" {
				builder.WriteString(arg.name)
				builder.WriteString("=")
			}

			builder.WriteString(arg.arg.String())
		}

		builder.WriteRune(')')
	}

	if v.isTemplateBlock {
		builder.WriteString(" }}")
	}

	return builder.String()
}
func (v *Variable) AddArgument(argName string, node AST) {
	v.argCall = append(v.argCall, funcCallArg{argName, node})
}

func (v *Variable) IsSimpleIdent(name string) bool {
	return v.varType == identVar && v.token.Value == name
}

func (v *Variable) wrap(wrappedVarType variableType) *Variable {
	nv := NewVariable(v.token)
	nv.varType = wrappedVarType
	nv.subVariable = v

	return nv
}

func (v *Variable) AsCallable() *Variable {
	return v.wrap(funcCallVar)
}

func (v *Variable) AsMapLookupTo(key AST) *Variable {
	nv := v.wrap(mapLookupVar)
	nv.lookupKey = key
	return nv
}

func (v *Variable) AsIndexTo(key *lexer.Token) *Variable {
	nv := v.wrap(indexVar)
	nv.token = key
	return nv
}

func (v *Variable) SetIsTemplateblock() {
	v.isTemplateBlock = true
}
