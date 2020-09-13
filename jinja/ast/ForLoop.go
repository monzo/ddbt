package ast

import (
	"fmt"

	"ddbt/compilerInterface"
	"ddbt/jinja/lexer"
)

// A block which represents a simple
type ForLoop struct {
	position     lexer.Position
	keyItrName   string
	valueItrName string
	list         *Variable
	body         *Body
}

type ForLoopParameter struct {
	name         string
	defaultValue *lexer.Token
}

var _ AST = &ForLoop{}

func NewForLoop(valueItrToken *lexer.Token, keyItr string, list *Variable) *ForLoop {
	return &ForLoop{
		position:     valueItrToken.Start,
		keyItrName:   keyItr,
		valueItrName: valueItrToken.Value,
		list:         list,
		body:         NewBody(valueItrToken),
	}
}

func (fl *ForLoop) Position() lexer.Position {
	return fl.position
}

func (fl *ForLoop) Execute(ec compilerInterface.ExecutionContext) (compilerInterface.AST, error) {
	// TODO add a map variable called "loop" with a "last" bool param
	return nil, nil
}

func (fl *ForLoop) String() string {
	if fl.keyItrName != "" {
		return fmt.Sprintf("\n{%% for %s, %s in %s %%}%s{%% endfor %%}", fl.keyItrName, fl.valueItrName, fl.list.String(), fl.body.String())
	} else {
		return fmt.Sprintf("\n{%% for %s in %s %%}%s{%% endfor %%}", fl.valueItrName, fl.list.String(), fl.body.String())
	}
}

func (fl *ForLoop) AppendBody(node AST) {
	fl.body.Append(node)
}
