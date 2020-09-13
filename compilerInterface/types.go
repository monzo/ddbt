package compilerInterface

import "ddbt/jinja/lexer"

type ExecutionContext interface {
	SetVariable(name string, value *Value)
	GetVariable(name string) *Value
	ErrorAt(part AST, error string) error
}

type AST interface {
	Execute(ec ExecutionContext) (*Value, error)
	Position() lexer.Position
	String() string
}

type ValueType string

const (
	Undefined     ValueType = "undefined"
	NullVal       ValueType = "null"
	StringVal     ValueType = "String"
	NumberVal     ValueType = "Number"
	MapVal        ValueType = "Map"
	ListVal       ValueType = "List"
	FunctionalVal ValueType = "Function"
)

type Value struct {
	StringValue string
	NumberValue float64
	MapValue    map[string]*Value
	ListValue   []*Value
	Function    AST
	IsUndefined bool
	IsNull      bool
}

func (v *Value) Type() ValueType {
	switch {
	case v.IsUndefined:
		return Undefined

	case v.IsNull:
		return NullVal

	case v.MapValue != nil:
		return MapVal

	case v.ListValue != nil:
		return ListVal

	case v.NumberValue != 0:
		return NumberVal

	case v.StringValue != "":
		return StringVal

	case v.Function != nil:
		// Note: function call is last so that if a user overrides a function
		// with a value, we could still the original function/macro
		return FunctionalVal

	default:
		// Incase of "" as the value
		return StringVal
	}
}
