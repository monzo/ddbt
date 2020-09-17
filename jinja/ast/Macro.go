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
			// quick lookup map
			namedArgs := make(map[string]*compilerInterface.Value)
			for _, arg := range args {
				if arg.Name != "" {
					namedArgs[arg.Name] = arg.Value
				}
			}

			stillOrdered := true

			// Process all the parameters, checking what args where provided
			for i, param := range m.parameters {
				if value, found := namedArgs[param.name]; found {
					ec.SetVariable(param.name, value)

					stillOrdered = len(args) > i && args[i].Name == param.name
				} else if len(args) <= i {
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
				}
			}

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
