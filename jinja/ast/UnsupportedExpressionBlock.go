package ast

import "ddbt/jinja/lexer"

type UnsupportedExpressionBlock struct {
	position lexer.Position
}

var _ AST = &UnsupportedExpressionBlock{}

func NewUnsupportedExpressionBlock(token *lexer.Token) *UnsupportedExpressionBlock {
	return &UnsupportedExpressionBlock{
		position: token.Start,
	}
}

func (b *UnsupportedExpressionBlock) Position() lexer.Position {
	return b.position
}

func (b *UnsupportedExpressionBlock) Execute(_ *ExecutionContext) AST {
	return nil
}

func (b *UnsupportedExpressionBlock) String() string {
	return ""
}

func (b *UnsupportedExpressionBlock) AppendBody(node AST) {
	// no-op
}
