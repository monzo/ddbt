package compilerInterface

import (
	"ddbt/config"
	"ddbt/jinja/lexer"
)

type ExecutionContext interface {
	SetVariable(name string, value *Value)
	GetVariable(name string) *Value

	ErrorAt(part AST, error string) error
	NilResultFor(part AST) error
	PushState() ExecutionContext
	CopyVariablesInto(ec ExecutionContext)

	RegisterMacro(name string, ec ExecutionContext, function FunctionDef)
	RegisterUpstreamAndGetRef(name string, fileType string) (*Value, error)

	FileName() string
	GetTarget() (*config.Target, error)
	MarkAsDynamicSQL() (*Value, error)
}

type AST interface {
	Execute(ec ExecutionContext) (*Value, error)
	Position() lexer.Position
	String() string
}

type Argument struct {
	Name  string // optional
	Value *Value
}

type Arguments []Argument

func (args Arguments) ToVarArgs() *Value {
	varargs := make([]*Value, len(args))

	for i, value := range args {
		varargs[i] = value.Value
	}

	return NewList(varargs)
}

type FunctionDef func(ec ExecutionContext, caller AST, args Arguments) (*Value, error)
