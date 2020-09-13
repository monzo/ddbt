package ast

import (
	"ddbt/compilerInterface"
)

type AST compilerInterface.AST

type BodyHoldingAST interface {
	AST
	AppendBody(node AST)
}

type ArgumentHoldingAST interface {
	AST
	AddArgument(argName string, node AST)
}
