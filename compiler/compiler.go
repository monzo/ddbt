package compiler

import (
	"errors"
	"fmt"
	"strings"

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

func CompileModel(file *fs.File, gc *GlobalContext, isExecuting bool) error {
	ec := NewExecutionContext(file, gc.fileSystem, isExecuting, gc, gc)

	target, err := file.GetTarget()
	if err != nil {
		return err
	}

	ec.SetVariable("this", compilerInterface.NewMap(map[string]*compilerInterface.Value{
		"schema": compilerInterface.NewString(target.DataSet),
		"table":  compilerInterface.NewString(file.Name),
		"name":   compilerInterface.NewString(file.Name),
	}))

	ec.SetVariable("target", compilerInterface.NewMap(map[string]*compilerInterface.Value{
		"name":    compilerInterface.NewString(config.GlobalCfg.Target.Name),
		"schema":  compilerInterface.NewString(target.DataSet),
		"dataset": compilerInterface.NewString(target.DataSet),
		"type":    compilerInterface.NewString("bigquery"),
		"threads": compilerInterface.NewNumber(float64(target.Threads)),
		"project": compilerInterface.NewString(target.ProjectID),
	}))

	ec.SetVariable("config", file.ConfigObject())
	ec.SetVariable("execute", compilerInterface.NewBoolean(isExecuting))

	if file.SyntaxTree == nil {
		return errors.New(fmt.Sprintf("file %s has not been parsed before we attempt the compile", file.Name))
	}

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

	if len(file.EphemeralCTES) > 0 {
		var builder strings.Builder

		builder.WriteString("WITH ")

		for name, model := range file.EphemeralCTES {
			builder.WriteString(name)
			builder.WriteString(" AS (\n\t")
			model.Mutex.Lock()
			builder.WriteString(strings.Replace(strings.TrimSpace(model.CompiledContents), "\n", "\n\t", -1))
			model.Mutex.Unlock()
			builder.WriteString("\n),\n\n")
		}

		builder.WriteString("__dbt__main_query AS (\n\t")
		builder.WriteString(strings.Replace(strings.TrimSpace(file.CompiledContents), "\n", "\n\t", -1))
		builder.WriteString("\n)\n\nSELECT * FROM __dbt__main_query")

		file.CompiledContents = builder.String()
	}

	file.Mutex.Unlock()

	return err
}

func CompileStringWithCache(s string) (string, error) {
	fileSystem, err := fs.InMemoryFileSystem(map[string]string{"models/____": s})
	if err != nil {
		return "", err
	}

	model := fileSystem.Model("____")
	err = ParseFile(model)
	if err != nil {
		return "", err
	}

	gc, err := NewGlobalContext(config.GlobalCfg, fileSystem)
	if err != nil {
		return "", err
	}

	for _, file := range fileSystem.Macros() {
		if err := CompileModel(file, gc, false); err != nil {
			return "", err
		}
	}
	ec := NewExecutionContext(model, fileSystem, true, gc, gc)

	finalValue, err := model.SyntaxTree.Execute(ec)
	if err != nil {
		return "", err
	}

	return finalValue.AsStringValue(), nil
}

func isOnlyCompilingSQL(ec compilerInterface.ExecutionContext) bool {
	value := ec.GetVariable("execute")

	if value.Type() == compilerInterface.BooleanValue {
		return !value.BooleanValue
	} else {
		return true
	}
}
