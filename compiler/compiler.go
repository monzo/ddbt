package compiler

import (
	"errors"

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

	return finalAST.String(), err
}
