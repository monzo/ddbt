package compiler

import (
	"errors"

	"ddbt/compilerInterface"
	"ddbt/config"
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

func CompileModel(file *fs.File, gc *GlobalContext) error {
	ec := NewExecutionContext(file, gc.fileSystem, gc)

	ec.SetVariable("this", compilerInterface.NewMap(map[string]*compilerInterface.Value{
		"schema": compilerInterface.NewString(config.GlobalCfg.Target.DataSet),
		"table":  compilerInterface.NewString(file.Name),
		"name":   compilerInterface.NewString(file.Name),
	}))

	ec.SetVariable("config", file.ConfigObject())

	finalAST, err := file.SyntaxTree.Execute(ec)
	if err != nil {
		return err
	}

	if finalAST == nil {
		return errors.New("no AST returned after execution")
	}

	if finalAST.Type() != compilerInterface.StringVal && finalAST.Type() != compilerInterface.ReturnVal {
		return errors.New("AST did not return a string")
	}

	file.Mutex.Lock()
	file.CompiledContents = finalAST.AsStringValue()
	file.Mutex.Unlock()

	return err
}
