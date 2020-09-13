package compiler

import (
	"errors"

	"ddbt/compilerInterface"
	"ddbt/fs"
)

func CompileModel(file *fs.File) (string, error) {
	ec := NewExecutionContext()

	finalAST, err := file.SyntaxTree.Execute(ec)
	if err != nil {
		return "", err
	}

	if finalAST == nil {
		return "", errors.New("no AST returned after execution")
	}

	if finalAST.Type() != compilerInterface.StringVal {
		return "", errors.New("AST did not return a string")
	}

	return finalAST.StringValue, err
}
