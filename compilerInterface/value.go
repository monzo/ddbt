package compilerInterface

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"ddbt/jinja/lexer"
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
	ReturnVal     ValueType = "ReturnVal" // marker to shortcut the rest of the file
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
	ReturnValue  *Value
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

func NewMap(data map[string]*Value) *Value {
	return &Value{ValueType: MapVal, MapValue: data}
}

func NewList(data []*Value) *Value {
	return &Value{ValueType: ListVal, ListValue: data}
}

func NewStringList(data []string) *Value {
	l := make([]*Value, len(data))

	for i, s := range data {
		l[i] = NewString(s)
	}

	return NewList(l)
}

func NewFunction(f FunctionDef) *Value {
	return &Value{
		ValueType: FunctionalVal,
		Function:  f,
	}
}

func NewUndefined() *Value {
	return &Value{
		ValueType:   Undefined,
		IsUndefined: true,
	}
}

func NewReturnValue(value *Value) *Value {
	return &Value{
		ValueType:   ReturnVal,
		ReturnValue: value,
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
			"items": NewFunction(func(_ ExecutionContext, _ AST, _ Arguments) (*Value, error) { return v, nil }),
			"extend": NewFunction(func(_ ExecutionContext, _ AST, args Arguments) (*Value, error) {
				for _, arg := range args[0].Value.ListValue {
					if arg != v {
						v.ListValue = append(v.ListValue, arg)
					}
				}
				return v, nil
			}),
		}

	case ReturnVal:
		return v.ReturnValue.Properties()

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

	case ReturnVal:
		return v.ReturnValue.TruthyValue()

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
		return strconv.FormatFloat(v.NumberValue, 'f', -1, 64)

	case StringVal:
		return v.StringValue

	case ListVal:
		return fmt.Sprintf("%p", v.ListValue)

	case MapVal:
		return fmt.Sprintf("%p", v.MapValue)

	case NullVal, Undefined:
		return ""

	case ReturnVal:
		return v.ReturnValue.AsStringValue()

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

	case ReturnVal:
		return v.ReturnValue.AsNumberValue()

	default:
		return 0, errors.New(fmt.Sprintf("unable to convert %s to number", v.Type()))
	}
}

func (v *Value) Unwrap() *Value {
	if v.ValueType == ReturnVal {
		return v.ReturnValue
	} else {
		return v
	}
}

func (v *Value) Equals(other *Value) bool {
	if v.ValueType == ReturnVal {
		return v.ReturnValue.Equals(other)
	}

	vType := v.Type()

	other = other.Unwrap()

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

func ValueFromToken(t *lexer.Token) (*Value, error) {
	switch t.Type {

	case lexer.StringToken:
		return NewString(t.Value), nil

	case lexer.NumberToken:
		f, err := strconv.ParseFloat(t.Value, 64)
		if err != nil {
			return nil, err
		}
		return NewNumber(f), nil

	case lexer.TrueToken:
		return NewBoolean(true), nil

	case lexer.FalseToken:
		return NewBoolean(false), nil

	case lexer.NullToken:
		return &Value{ValueType: NullVal, IsNull: true}, nil

	case lexer.NoneToken:
		return NewUndefined(), nil

	case lexer.IdentToken:
		return nil, errors.New("unable to convert ident to value: " + t.Value)

	default:
		return nil, errors.New(fmt.Sprintf("unable to convert %s to value", t.Type))
	}
}

func NewValueFromInterface(value interface{}) (*Value, error) {
	switch v := value.(type) {
	case string:
		return NewString(v), nil
	case int:
		return NewNumber(float64(v)), nil
	case int64:
		return NewNumber(float64(v)), nil
	case uint:
		return NewNumber(float64(v)), nil
	case uint64:
		return NewNumber(float64(v)), nil
	case float32:
		return NewNumber(float64(v)), nil
	case float64:
		return NewNumber(v), nil
	case bool:
		return NewBoolean(v), nil

	default:
		return nil, errors.New(fmt.Sprintf("Unknown value type %v", reflect.TypeOf(value)))
	}
}
