package compilerInterface

import "ddbt/jinja/lexer"

type ExecutionContext interface {
	SetVariable(name string, value *Variable)
	GetVariable(name string) *Variable
	ErrorAt(part AST, error string) error
}

type AST interface {
	Execute(ec ExecutionContext) (AST, error)
	Position() lexer.Position
	String() string
}

type VariableType string

const (
	Undefined VariableType = "undefined"
	StringVar              = "String"
	NumberVar              = "Number"
	MapVar                 = "Map"
	ListVar                = "List"
)

type Variable struct {
	StringValue string
	NumberValue float64
	MapValue    map[string]*Variable
	ListValue   []*Variable
	IsUndefined bool
}

func (v *Variable) Type() VariableType {
	switch {
	case v.IsUndefined:
		return Undefined
	case v.MapValue != nil:
		return MapVar
	case v.ListValue != nil:
		return ListVar
	case v.NumberValue != 0:
		return NumberVar
	default:
		return StringVar
	}
}
