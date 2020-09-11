package ast

import (
	"fmt"
	"strings"

	"ddbt/jinja/lexer"
)

// A block which represents a simple
type Macro struct {
	position   lexer.Position
	name       string
	body       *Body
	parameters []macroParameter
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

func (m *Macro) Execute(_ *ExecutionContext) AST {
	return nil
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

			if param.defaultValue.Type == lexer.StringToken {
				builder.WriteRune('\'')
			}

			builder.WriteString(param.defaultValue.Value)

			if param.defaultValue.Type == lexer.StringToken {
				builder.WriteRune('\'')
			}
		}
	}

	return fmt.Sprintf("\n{%% macro %s(%s) %%}%s{%% endmacro %%}", m.name, builder.String(), m.body.String())
}

func (m *Macro) AddParameter(name string, defaultValue *lexer.Token) {
	m.parameters = append(
		m.parameters,
		macroParameter{name, defaultValue},
	)
}

func (m *Macro) AppendBody(node AST) {
	m.body.Append(node)
}
