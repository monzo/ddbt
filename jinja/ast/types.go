package ast

import "ddbt/jinja/lexer"

type AST interface {
	Position() lexer.Position
	Execute(ec *ExecutionContext) AST
	String() string
}

type BodyHoldingAST interface {
	AST
	AppendBody(node AST)
}

type ArgumentHoldingAST interface {
	AST
	AddArgument(argName string, node AST)
}
