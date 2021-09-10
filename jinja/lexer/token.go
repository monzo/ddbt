package lexer

import "fmt"

type Position struct {
	File   string
	Column int
	Row    int
}

type TokenType string

const (
	ErrorToken TokenType = "ERR"
	EOFToken   TokenType = "EOF"

	// Token types in the raw text blocks
	TextToken                TokenType = "TEXT"
	TemplateBlockOpen        TokenType = "{{"
	TemplateBlockClose       TokenType = "}}"
	ExpressionBlockOpen      TokenType = "{%"
	ExpressionBlockOpenTrim  TokenType = "{%-"
	ExpressionBlockClose     TokenType = "%}"
	ExpressionBlockCloseTrim TokenType = "-%}"

	// Token types purely within the code blocks
	IdentToken             TokenType = "IDENT"
	LeftParenthesesToken   TokenType = "("
	RightParenthesesToken  TokenType = ")"
	LeftBracketToken       TokenType = "["
	RightBracketToken      TokenType = "]"
	LeftBraceToken         TokenType = "{"
	RightBraceToken        TokenType = "}"
	EqualsToken            TokenType = "="
	IsEqualsToken          TokenType = "=="
	NotEqualsToken         TokenType = "!="
	LessThanEqualsToken    TokenType = "<="
	GreaterThanEqualsToken TokenType = ">="
	LessThanToken          TokenType = "<"
	GreaterThanToken       TokenType = ">"
	ColonToken             TokenType = ":"
	StringToken            TokenType = "STRING"
	NumberToken            TokenType = "NUMBER"
	CommaToken             TokenType = ","
	PeriodToken            TokenType = "."
	MinusToken             TokenType = "-"
	PlusToken              TokenType = "+"
	MultiplyToken          TokenType = "*"
	PowerToken             TokenType = "**"
	DivideToken            TokenType = "/"
	PipeToken              TokenType = "|"
	TildeToken             TokenType = "~"
	TrueToken              TokenType = "TRUE"
	FalseToken             TokenType = "FALSE"
	NullToken              TokenType = "NULL"
	NoneToken              TokenType = "None"
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
