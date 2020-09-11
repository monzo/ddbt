package ast

import (
	"fmt"

	"ddbt/jinja/lexer"
)

// An special marker to tracking if we're at the end of a parse block
type AtomExpressionBlock struct {
	token *lexer.Token
}

var _ AST = &AtomExpressionBlock{}

func NewAtomExpressionBlock(token *lexer.Token) *AtomExpressionBlock {
	return &AtomExpressionBlock{
		token: token,
	}
}

func (a *AtomExpressionBlock) Position() lexer.Position {
	return a.token.Start
}

func (a *AtomExpressionBlock) Execute(_ *ExecutionContext) AST {
	return nil
}

func (a *AtomExpressionBlock) String() string {
	return fmt.Sprintf("{%% %s %%}", a.token.Value)
}

func (a *AtomExpressionBlock) Token() *lexer.Token {
	return a.token
}
