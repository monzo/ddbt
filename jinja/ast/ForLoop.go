package ast

import (
	"fmt"

	"ddbt/jinja/lexer"
)

// A block which represents a simple
type ForLoop struct {
	position     lexer.Position
	iteratorName string
	list         *Variable
	body         *Body
}

type ForLoopParameter struct {
	name         string
	defaultValue *lexer.Token
}

var _ AST = &ForLoop{}

func NewForLoop(iteratorToken *lexer.Token, list *Variable) *ForLoop {
	return &ForLoop{
		position:     iteratorToken.Start,
		iteratorName: iteratorToken.Value,
		list:         list,
		body:         NewBody(iteratorToken),
	}
}

func (fl *ForLoop) Position() lexer.Position {
	return fl.position
}

func (fl *ForLoop) Execute(_ *ExecutionContext) AST {
	// TODO add a map variable called "loop" with a "last" bool param
	return nil
}

func (fl *ForLoop) String() string {
	return fmt.Sprintf("\n{%% for %s in %s %%}%s{%% endfor %%}", fl.iteratorName, fl.list.String(), fl.body.String())
}

func (fl *ForLoop) AppendBody(node AST) {
	fl.body.Append(node)
}
