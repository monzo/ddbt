package ast

import (
	"errors"
	"fmt"
	"strings"

	"ddbt/compilerInterface"
	"ddbt/jinja/lexer"
)

// A block which represents a simple
type Macro struct {
	position          lexer.Position
	name              string
	body              *Body
	parameters        []macroParameter
	numOptionalParams int
}

type macroParameter struct {
	name         string
	defaultValue *lexer.Token
}

var _ AST = &Macro{}

func NewMacro(token *lexer.Token) *Macro {
	return &Macro{
		position:   token.Start,
		name:       token.Value,
		body:       NewBody(token),
		parameters: make([]macroParameter, 0),
	}
}

func (m *Macro) Position() lexer.Position {
	return m.position
}

func (m *Macro) Execute(macroEC compilerInterface.ExecutionContext) (*compilerInterface.Value, error) {
	macroEC.RegisterMacro(
		m.name,
		macroEC,
		func(ec compilerInterface.ExecutionContext, caller compilerInterface.AST, args compilerInterface.Arguments) (*compilerInterface.Value, error) {
			unusedArguments := make(compilerInterface.Arguments, 0) // Unused positional based arguments
			kwargs := make(map[string]*compilerInterface.Value)     // Unused name based arguments

			// Init both varargs and kwargs to all arguments passed in
			for _, arg := range args {
				if arg.Name != "" {
					kwargs[arg.Name] = arg.Value
				}

				unusedArguments = append(unusedArguments, arg)
			}

			stillOrdered := true

			// Process all the parameters, checking what args where provided
			for i, param := range m.parameters {
				if value, found := kwargs[param.name]; found {
					delete(kwargs, param.name) // Consume the used keyword arg
					ec.SetVariable(param.name, value)

					stillOrdered = len(args) > i && args[i].Name == param.name

					if stillOrdered {
						unusedArguments = unusedArguments[1:] // this positional argument has been used
					}
				} else if len(args) <= i || args[i].Name != "" {
					stillOrdered = false

					if param.defaultValue != nil {
						value, err := compilerInterface.ValueFromToken(param.defaultValue)
						if err != nil {
							return nil, ec.ErrorAt(caller, fmt.Sprintf("Unable to understand default value for %s: %s", param.name, err))
						}
						ec.SetVariable(param.name, value)
					} else {
						ec.SetVariable(param.name, compilerInterface.NewUndefined())
					}
				} else if !stillOrdered {
					return nil, ec.ErrorAt(caller, fmt.Sprintf("Named arguments have been used out of order, please either used all named arguments or keep them in order. Unable to identify what %s should be.", param.name))
				} else {
					ec.SetVariable(param.name, args[i].Value)
					unusedArguments = unusedArguments[1:] // this positional argument has been used
				}
			}

			// Now take all the remaining unnamed arguments, and place them in the varargs
			varArgs := make(compilerInterface.Arguments, 0)
			for _, arg := range unusedArguments {
				if arg.Name == "" {
					varArgs = append(varArgs, arg)
				}
			}

			ec.SetVariable("varargs", varArgs.ToVarArgs())
			ec.SetVariable("kwargs", compilerInterface.NewMap(kwargs))

			result, err := m.body.Execute(ec)
			if err != nil {
				return nil, err
			}
			if result == nil {
				return nil, ec.NilResultFor(caller)
			}

			return result.Unwrap(), err
		},
	)

	return &compilerInterface.Value{IsUndefined: true}, nil
}

func (m *Macro) String() string {
	var builder strings.Builder

	for i, param := range m.parameters {
		if i > 0 {
			builder.WriteString(", ")
		}

		builder.WriteString(param.name)

		if param.defaultValue != nil {
			builder.WriteString(" = ")

			switch param.defaultValue.Type {
			case lexer.StringToken:
				builder.WriteRune('\'')
				builder.WriteString(param.defaultValue.Value)
				builder.WriteRune('\'')

			case lexer.NumberToken:
				builder.WriteString(param.defaultValue.Value)

			case lexer.TrueToken:
				builder.WriteString("TRUE")

			case lexer.FalseToken:
				builder.WriteString("FALSE")
			}
		}
	}

	return fmt.Sprintf("\n{%% macro %s(%s) %%}%s{%% endmacro %%}", m.name, builder.String(), m.body.String())
}

func (m *Macro) AddParameter(name string, defaultValue *lexer.Token) error {
	if defaultValue != nil {
		m.numOptionalParams++
	} else if m.numOptionalParams > 0 {
		return errors.New("can not have non-operation parameter after an optional one")
	}

	m.parameters = append(
		m.parameters,
		macroParameter{name, defaultValue},
	)

	return nil
}

func (m *Macro) AppendBody(node AST) {
	m.body.Append(node)
}
