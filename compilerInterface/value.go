package compilerInterface

import (
	"errors"
	"fmt"
	"strconv"
)

type ValueType string

const (
	Dynamic       ValueType = ""
	Undefined     ValueType = "undefined"
	NullVal       ValueType = "null"
	StringVal     ValueType = "String"
	NumberVal     ValueType = "Number"
	BooleanValue  ValueType = "Boolean"
	MapVal        ValueType = "Map"
	ListVal       ValueType = "List"
	FunctionalVal ValueType = "Function"
)

type Value struct {
	ValueType    ValueType
	StringValue  string
	NumberValue  float64
	BooleanValue bool
	MapValue     map[string]*Value
	ListValue    []*Value
	Function     FunctionDef
	IsUndefined  bool
	IsNull       bool
}

func NewBoolean(value bool) *Value {
	return &Value{ValueType: BooleanValue, BooleanValue: value}
}

func NewString(value string) *Value {
	return &Value{ValueType: StringVal, StringValue: value}
}

func NewNumber(value float64) *Value {
	return &Value{ValueType: NumberVal, NumberValue: value}
}

func NewFunction(f FunctionDef) *Value {
	return &Value{
		ValueType: FunctionalVal,
		Function:  f,
	}
}

func (v *Value) Type() ValueType {
	if v == nil {
		return NullVal
	}

	if v.ValueType != Dynamic {
		return v.ValueType
	}

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

func (v *Value) Properties() map[string]*Value {
	switch v.Type() {
	case MapVal:
		return v.MapValue

	case ListVal:
		return map[string]*Value{
			"items": NewFunction(func(_ ExecutionContext, _ Arguments) (*Value, error) { return v, nil }),
		}

	default:
		return nil
	}
}

func (v *Value) TruthyValue() bool {
	switch v.Type() {
	case BooleanValue:
		return v.BooleanValue

	case NumberVal:
		return v.NumberValue != 0

	case StringVal:
		return v.StringValue != ""

	case ListVal:
		return len(v.ListValue) > 0

	case MapVal:
		return len(v.MapValue) > 0

	case NullVal, Undefined:
		return false

	default:
		panic("Unable to truthy " + v.Type())
	}
}

func (v *Value) AsStringValue() string {
	switch v.Type() {
	case BooleanValue:
		if v.BooleanValue {
			return "TRUE"
		} else {
			return "FALSE"
		}

	case NumberVal:
		return fmt.Sprintf("%g", v.NumberValue)

	case StringVal:
		return v.StringValue

	case ListVal:
		return fmt.Sprintf("%p", v.ListValue)

	case MapVal:
		return fmt.Sprintf("%p", v.ListValue)

	case NullVal, Undefined:
		return ""

	default:
		panic("Unable to truthy " + v.Type())
	}
}

func (v *Value) AsNumberValue() (float64, error) {
	switch v.Type() {
	case BooleanValue:
		if v.BooleanValue {
			return 1, nil
		} else {
			return 0, nil
		}

	case NumberVal:
		return v.NumberValue, nil

	case StringVal:
		return strconv.ParseFloat(v.StringValue, 64)

	case NullVal, Undefined:
		return 0, nil

	default:
		return 0, errors.New(fmt.Sprintf("unable to convert %s to number", v.Type()))
	}
}

func (v *Value) Equals(other *Value) bool {
	vType := v.Type()

	if vType != other.Type() {
		return false
	}

	switch vType {
	case BooleanValue:
		return v.BooleanValue == other.BooleanValue

	case NumberVal:
		return v.NumberValue == other.NumberValue

	case StringVal:
		return v.StringValue == other.StringValue

	case ListVal:
		if len(v.ListValue) != len(other.ListValue) {
			return false
		}

		for i, value := range v.ListValue {
			if !value.Equals(other.ListValue[i]) {
				return false
			}
		}

		return true

	case MapVal:
		if len(v.MapValue) != len(other.MapValue) {
			return false
		}

		for key, value := range v.MapValue {
			if !value.Equals(other.MapValue[key]) {
				return false
			}
		}

		return true

	case NullVal, Undefined:
		return true

	default:
		panic("Unable to compare value types " + vType)
	}
}

func (v *Value) String() string {
	return fmt.Sprintf("%s(%s)", v.Type(), v.AsStringValue())
}
