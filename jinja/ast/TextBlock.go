package ast

import (
	"strings"
	"unicode"

	"ddbt/compilerInterface"
	"ddbt/jinja/lexer"
)

// A block which represents a simple
type TextBlock struct {
	position lexer.Position
	value    string
}

var _ AST = &TextBlock{}

func NewTextBlock(token *lexer.Token) *TextBlock {
	return &TextBlock{
		position: token.Start,
		value:    token.Value,
	}
}

func newTextBlockAt(position lexer.Position, text string) *TextBlock {
	return &TextBlock{
		position: position,
		value:    text,
	}
}

func (tb *TextBlock) Position() lexer.Position {
	return tb.position
}

func (tb *TextBlock) Execute(_ compilerInterface.ExecutionContext) (compilerInterface.AST, error) {
	return tb, nil // no-op text blocks don't compile to anything apart from themselves
}

func (tb *TextBlock) String() string {
	return tb.value
}

func (tb *TextBlock) TrimPrefixWhitespace() string {
	tb.value = strings.TrimLeftFunc(tb.value, unicode.IsSpace)

	return tb.value
}
