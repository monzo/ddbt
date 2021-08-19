package ast

import (
	"fmt"
	"strings"

	"ddbt/compilerInterface"
	"ddbt/jinja/lexer"
)

type variableType = string

const (
	identVar          variableType = "IDENT"
	propertyLookupVar              = "PROPERTY_LOOKUP"
	indexLookupVar                 = "INDEX_LOOKUP"
	funcCallVar                    = "FUNC_CALL"
)

type Variable struct {
	token       *lexer.Token
	varType     variableType
	subVariable *Variable

	argCall         funcCallArgs
	lookupKey       AST
	isTemplateBlock bool
}

var _ AST = &Variable{}

func NewVariable(token *lexer.Token) *Variable {
	return &Variable{
		token:   token,
		varType: identVar,
		argCall: make(funcCallArgs, 0),
	}
}

func (v *Variable) Position() lexer.Position {
	return v.token.Start
}

func (v *Variable) Execute(ec compilerInterface.ExecutionContext) (*compilerInterface.Value, error) {
	variable, err := v.resolve(ec, false)
	if err != nil {
		return nil, err
	}

	if variable == nil {
		return nil, ec.ErrorAt(v, "nil variable received after resolve")
	} else {
		return variable, nil
	}
}

func (v *Variable) resolve(ec compilerInterface.ExecutionContext, isForFunctionCall bool) (*compilerInterface.Value, error) {
	switch v.varType {
	case identVar:
		return ec.GetVariable(v.token.Value), nil

	case indexLookupVar:
		return v.resolveIndexLookup(ec, isForFunctionCall)

	case propertyLookupVar:
		return v.resolvePropertyLookup(ec, isForFunctionCall)

	case funcCallVar:
		return v.resolveFunctionCall(ec)

	default:
		return nil, ec.ErrorAt(v, fmt.Sprintf("unable to resolve variable type %s: not implemented", v.varType))
	}
}

func (v *Variable) resolveIndexLookup(ec compilerInterface.ExecutionContext, isForFunctionCall bool) (*compilerInterface.Value, error) {
	value, err := v.subVariable.resolve(ec, false)
	if err != nil {
		return nil, err
	}

	lookupKey, err := v.lookupKey.Execute(ec)
	if err != nil {
		return nil, err
	}
	if lookupKey == nil {
		return nil, ec.NilResultFor(v.lookupKey)
	}

	t := value.Type()
	switch t {
	case compilerInterface.ListVal:
		lt := lookupKey.Type()
		if lt != compilerInterface.NumberVal && !(lookupKey.Type() == compilerInterface.StringVal && lookupKey.StringValue == "") {
			return nil, ec.ErrorAt(v.lookupKey, fmt.Sprintf("Number required to index into a list, got %s", lt))
		}

		index := int(lookupKey.NumberValue)

		if index < 0 {
			return nil, ec.ErrorAt(v.lookupKey, fmt.Sprintf("index below 0, got: %d", index))
		}
		if index >= len(value.ListValue) {
			return nil, ec.ErrorAt(v.lookupKey, fmt.Sprintf("index larger than cap %d, got: %d", len(value.ListValue), index))
		}

		return value.ListValue[index], nil

	case compilerInterface.MapVal:
		lt := lookupKey.Type()
		if lt != compilerInterface.StringVal || lookupKey.StringValue == "" {
			return nil, ec.ErrorAt(v.lookupKey, fmt.Sprintf("String required to index into a map, got %s", lt))
		}

		rtnValue, found := value.MapValue[lookupKey.StringValue]
		if !found {
			if isForFunctionCall && lookupKey.StringValue == "items" {
				// If we're asking for the items of a map, return the map back
				return compilerInterface.NewFunction(func(_ compilerInterface.ExecutionContext, _ compilerInterface.AST, _ compilerInterface.Arguments) (*compilerInterface.Value, error) {
					return value, nil
				}), nil
			} else {
				return &compilerInterface.Value{IsUndefined: true}, nil
			}
		}
		return rtnValue, nil

	default:
		return nil, ec.ErrorAt(v, fmt.Sprintf("unable reference by index in a %s", t))
	}
}

func (v *Variable) resolvePropertyLookup(ec compilerInterface.ExecutionContext, isForFunctionCall bool) (*compilerInterface.Value, error) {
	value, err := v.subVariable.resolve(ec, isForFunctionCall)
	if err != nil {
		return nil, err
	}

	data := value.Properties(isForFunctionCall)
	if data == nil {
		return nil, ec.ErrorAt(v, fmt.Sprintf("unable reference by property key in a %s", value.Type()))
	}

	rtnValue, found := data[v.token.Value]
	if !found {
		if isForFunctionCall && v.token.Value == "items" {
			// If we're asking for the items of a map, return the map back
			return compilerInterface.NewFunction(func(_ compilerInterface.ExecutionContext, _ compilerInterface.AST, _ compilerInterface.Arguments) (*compilerInterface.Value, error) {
				return value, nil
			}), nil
		} else if isForFunctionCall && v.token.Value == "get" {
			return compilerInterface.NewFunction(func(ec compilerInterface.ExecutionContext, caller compilerInterface.AST, args compilerInterface.Arguments) (*compilerInterface.Value, error) {
				defaultValue := compilerInterface.NewUndefined()

				switch len(args) {
				case 1:
				case 2:
					defaultValue = args[1].Value.Unwrap()
				default:
					return nil, ec.ErrorAt(caller, fmt.Sprintf("expected 1 or 2 arguments, got %d", len(args)))
				}

				key := args[0].Value.Unwrap()
				keyStr := key.AsStringValue()
				if keyStr == "" {
					return nil, ec.ErrorAt(caller, "key in the call to get() cannot be blank")
				}

				v, found := data[keyStr]
				if !found {
					return defaultValue, nil
				} else {
					return v, nil
				}
			}), nil
		} else {
			return &compilerInterface.Value{IsUndefined: true}, nil
		}
	}

	return rtnValue, nil
}

func (v *Variable) resolveFunctionCall(ec compilerInterface.ExecutionContext) (*compilerInterface.Value, error) {
	value, err := v.subVariable.resolve(ec, true)
	if err != nil {
		return nil, err
	}

	if value.Type() != compilerInterface.FunctionalVal && value.Function == nil {
		return nil, ec.ErrorAt(v.subVariable, fmt.Sprintf("expected `%s` to be a callable function, got %s", v.subVariable.String(), value.Type()))
	}

	arguments, err := v.argCall.Execute(ec)
	if err != nil {
		return nil, err
	}

	result, err := value.Function(ec.PushState(), v, arguments)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, ec.NilResultFor(v.subVariable)
	}

	return result, nil
}

func (v *Variable) String() string {
	var builder strings.Builder

	if v.isTemplateBlock {
		builder.WriteString("{{ ")
	}

	switch v.varType {
	case identVar:
		builder.WriteString(v.token.Value)

	case propertyLookupVar:
		builder.WriteString(v.subVariable.String())
		builder.WriteRune('.')
		builder.WriteString(v.token.Value)

	case indexLookupVar:
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

func (v *Variable) AsIndexLookup(key AST) *Variable {
	nv := v.wrap(indexLookupVar)
	nv.lookupKey = key
	return nv
}

func (v *Variable) AsPropertyLookup(key *lexer.Token) *Variable {
	nv := v.wrap(propertyLookupVar)
	nv.token = key
	return nv
}

func (v *Variable) SetIsTemplateblock() {
	v.isTemplateBlock = true
}
