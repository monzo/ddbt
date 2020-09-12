package ast

import (
	"strings"
	"unicode"

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

func (tb *TextBlock) Position() lexer.Position {
	return tb.position
}

func (tb *TextBlock) Execute(_ *ExecutionContext) AST {
	return nil
}

func (tb *TextBlock) String() string {
	return tb.value
}

func (tb *TextBlock) TrimPrefixWhitespace() string {
	tb.value = strings.TrimLeftFunc(tb.value, unicode.IsSpace)

	return tb.value
}
