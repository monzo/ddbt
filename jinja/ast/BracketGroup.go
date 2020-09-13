package ast

import (
	"fmt"

	"ddbt/compilerInterface"
	"ddbt/jinja/lexer"
)

// Tracks where the user wrapped brackets around parts of the tree
// to force operation order
type BracketGroup struct {
	value AST
}

var _ AST = &BracketGroup{}

func NewBracketGroup(bracketValue AST) *BracketGroup {
	return &BracketGroup{
		value: bracketValue,
	}
}

func (bg *BracketGroup) Position() lexer.Position {
	return bg.value.Position()
}

func (bg *BracketGroup) Execute(ec compilerInterface.ExecutionContext) (*compilerInterface.Value, error) {
	return bg.value.Execute(ec)
}

func (bg *BracketGroup) String() string {
	return fmt.Sprintf("(%s)", bg.value.String())
}
