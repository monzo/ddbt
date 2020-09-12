package lexer

import "fmt"

type Position struct {
	Column int
	Row    int
}

type TokenType string

const (
	ErrorToken TokenType = "ERR"
	EOFToken             = "EOF"

	// Token types in the raw text blocks
	TextToken                = "TEXT"
	TemplateBlockOpen        = "{{"
	TemplateBlockClose       = "}}"
	ExpressionBlockOpen      = "{%"
	ExpressionBlockOpenTrim  = "{%-"
	ExpressionBlockClose     = "%}"
	ExpressionBlockCloseTrim = "-%}"

	// Token types purely within the code blocks
	IdentToken             = "IDENT"
	LeftParenthesesToken   = "("
	RightParenthesesToken  = ")"
	LeftBracketToken       = "["
	RightBracketToken      = "]"
	LeftBraceToken         = "{"
	RightBraceToken        = "}"
	EqualsToken            = "="
	IsEqualsToken          = "=="
	NotEqualsToken         = "!="
	LessThanEqualsToken    = "<="
	GreaterThanEqualsToken = ">="
	LessThanToken          = "<"
	GreaterThanToken       = ">"
	ColonToken             = ":"
	StringToken            = "STRING"
	NumberToken            = "NUMBER"
	CommaToken             = ","
	PeriodToken            = "."
	MinusToken             = "-"
	PlusToken              = "+"
	MultiplyToken          = "*"
	DivideToken            = "/"
	PipeToken              = "|"
	TildeToken             = "~"
	TrueToken              = "TRUE"
	FalseToken             = "FALSE"
	NullToken              = "NULL"
)

type Token struct {
	Type  TokenType
	Value string

	// Position
	Start Position
	End   Position
}

func (t *Token) DisplayString() string {
	if t.Value == "" {
		return fmt.Sprintf("Token(`%s`)", t.Type)
	} else {
		return fmt.Sprintf("Token(`%s`, `%s`)", t.Type, t.Value)
	}
}

func (t *Token) String() string {
	if t.Value == "" {
		return fmt.Sprintf("Token(`%s`) @ %d:%d", t.Type, t.Start.Row, t.Start.Column)
	} else {
		return fmt.Sprintf("Token(`%s`, `%s`) @ %d:%d", t.Type, t.Value, t.Start.Row, t.Start.Column)
	}
}
