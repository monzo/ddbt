package ast

import (
	"fmt"
	"reflect"
	"strconv"
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

func (v *Variable) Execute(ec compilerInterface.ExecutionContext) (compilerInterface.AST, error) {
	variable, err := v.resolve(ec)
	if err != nil {
		return nil, err
	}

	if variable == nil {
		return nil, ec.ErrorAt(v, "nil variable received after resolve")
	} else {
		t := variable.Type()
		switch t {
		case compilerInterface.StringVar:
			return newTextBlockAt(v.token.Start, variable.StringValue), nil

		case compilerInterface.NumberVar:
			return newTextBlockAt(v.token.Start, fmt.Sprintf("%.f", variable.NumberValue)), nil

		default:
			return nil, ec.ErrorAt(v, fmt.Sprintf("unable to convert return data type %s into AST", t))
		}
	}
}

func (v *Variable) resolve(ec compilerInterface.ExecutionContext) (*compilerInterface.Variable, error) {
	switch v.varType {
	case identVar:
		return ec.GetVariable(v.token.Value), nil

	case indexLookupVar:
		return v.resolveIndexLookup(ec)

	case propertyLookupVar:
		return v.resolvePropertyLookup(ec)

	default:
		return nil, ec.ErrorAt(v, fmt.Sprintf("unable to resolve variable type %s: not implemented", v.varType))
	}
}

func (v *Variable) resolveIndexLookup(ec compilerInterface.ExecutionContext) (*compilerInterface.Variable, error) {
	value, err := v.subVariable.resolve(ec)
	if err != nil {
		return nil, err
	}

	lookupKey, err := v.lookupKey.Execute(ec)
	if err != nil {
		return nil, err
	}
	if lookupKey == nil {
		return nil, ec.ErrorAt(v.lookupKey, fmt.Sprintf("lookup key execution returned nil from %s", reflect.TypeOf(v.lookupKey)))
	}
	lookupKeyStr := lookupKey.String()

	t := value.Type()
	switch t {
	case compilerInterface.ListVar:
		index, err := strconv.Atoi(lookupKeyStr)
		if err != nil {
			return nil, ec.ErrorAt(v, fmt.Sprintf("uable to convert index `%s` to number: %s", lookupKeyStr, err))
		}

		if index < 0 {
			return nil, ec.ErrorAt(v.lookupKey, fmt.Sprintf("index below 0, got: %d", index))
		}
		if index >= len(value.ListValue) {
			return nil, ec.ErrorAt(v.lookupKey, fmt.Sprintf("index larger than cap %d, got: %d", len(value.ListValue), index))
		}

		return value.ListValue[index], nil

	case compilerInterface.MapVar:
		rtnValue, found := value.MapValue[lookupKeyStr]
		if !found {
			return &compilerInterface.Variable{IsUndefined: true}, nil
		}
		return rtnValue, nil

	default:
		return nil, ec.ErrorAt(v, fmt.Sprintf("unable reference by index in a %s", t))
	}
}

func (v *Variable) resolvePropertyLookup(ec compilerInterface.ExecutionContext) (*compilerInterface.Variable, error) {
	value, err := v.subVariable.resolve(ec)
	if err != nil {
		return nil, err
	}

	t := value.Type()
	switch t {
	case compilerInterface.MapVar:
		rtnValue, found := value.MapValue[v.token.Value]
		if !found {
			return &compilerInterface.Variable{IsUndefined: true}, nil
		}
		return rtnValue, nil

	default:
		return nil, ec.ErrorAt(v, fmt.Sprintf("unable reference by property key in a %s", t))
	}
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
