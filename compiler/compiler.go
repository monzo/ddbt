package compiler

import (
	"errors"

	"ddbt/compilerInterface"
	"ddbt/fs"
	"ddbt/jinja"
)

func ParseFile(file *fs.File) error {
	file.Mutex.Lock()
	defer file.Mutex.Unlock()

	if file.SyntaxTree == nil {
		syntaxTree, err := jinja.Parse(file)
		if err != nil {
			return err
		}

		file.SyntaxTree = syntaxTree
	}
	return nil
}

func CompileModel(file *fs.File, gc *GlobalContext) (string, error) {
	ec := NewExecutionContext(gc)

	ec.SetVariable("this", compilerInterface.NewMap(map[string]*compilerInterface.Value{
		"schema": compilerInterface.NewString("FIXME-DATASET"), // FIXME
		"table":  compilerInterface.NewString(file.Name),
		"name":   compilerInterface.NewString(file.Name),
	}))

	ec.SetVariable("config", file.ConfigObject())

	finalAST, err := file.SyntaxTree.Execute(ec)
	if err != nil {
		return "", err
	}

	if finalAST == nil {
		return "", errors.New("no AST returned after execution")
	}

	if finalAST.Type() != compilerInterface.StringVal && finalAST.Type() != compilerInterface.ReturnVal {
		return "", errors.New("AST did not return a string")
	}

	return finalAST.AsStringValue(), err
}
