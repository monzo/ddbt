package ast

import (
	"ddbt/compilerInterface"
	"ddbt/jinja/lexer"
)

// A block which represents a simple
type EndOfFile struct {
	position lexer.Position
}

var _ AST = &EndOfFile{}

func NewEndOfFile(token *lexer.Token) *EndOfFile {
	return &EndOfFile{
		position: token.Start,
	}
}

func (eof *EndOfFile) Position() lexer.Position {
	return eof.position
}

func (eof *EndOfFile) Execute(_ compilerInterface.ExecutionContext) (compilerInterface.AST, error) {
	return newTextBlockAt(eof.position, ""), nil
}

func (eof *EndOfFile) String() string {
	return ""
}
