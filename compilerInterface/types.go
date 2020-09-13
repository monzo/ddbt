package compilerInterface

import "ddbt/jinja/lexer"

type ExecutionContext interface {
	SetVariable(name string, value *Value)
	GetVariable(name string) *Value

	ErrorAt(part AST, error string) error
	NilResultFor(part AST) error
	PushState()
	PopState()
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

type FunctionDef func(ec ExecutionContext, args Arguments) (*Value, error)
