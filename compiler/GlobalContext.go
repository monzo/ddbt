package compiler

import (
	"errors"
	"fmt"
	"sync"

	"ddbt/compilerInterface"
	"ddbt/fs"
	"ddbt/utils"
)

type GlobalContext struct {
	fileSystem *fs.FileSystem

	macroMutex sync.RWMutex
	macros     map[string]*macroDef

	constants map[string]*compilerInterface.Value
}

type macroDef struct {
	ec       compilerInterface.ExecutionContext
	function compilerInterface.FunctionDef
}

var _ compilerInterface.ExecutionContext = &GlobalContext{}

func NewGlobalContext(fileSystem *fs.FileSystem) *GlobalContext {
	return &GlobalContext{
		fileSystem: fileSystem,
		macros:     make(map[string]*macroDef),
		constants: map[string]*compilerInterface.Value{
			"adapter": funcMapAsValue(adapterFunctions),

			"exceptions": funcMapAsValue(funcMap{
				"raise_compiler_error": func(ec compilerInterface.ExecutionContext, caller compilerInterface.AST, args compilerInterface.Arguments) (*compilerInterface.Value, error) {
					err := "error raised"

					if len(args) > 0 {
						err = args[0].Value.AsStringValue()
					}

					return nil, ec.ErrorAt(caller, err)
				},

				"warn": func(ec compilerInterface.ExecutionContext, caller compilerInterface.AST, args compilerInterface.Arguments) (*compilerInterface.Value, error) {
					err := "warning raised"

					if len(args) > 0 {
						err = args[0].Value.AsStringValue()
					}

					fmt.Printf("\n\nWARN: %s @ %s:%d:%d\n\n", err, caller.Position().File, caller.Position().Row, caller.Position().Column)

					return compilerInterface.NewUndefined(), nil
				},
			}),

			// We are always executing (https://docs.getdbt.com/reference/dbt-jinja-functions/execute)
			"execute": compilerInterface.NewBoolean(true),

			// https://docs.getdbt.com/reference/dbt-jinja-functions/modules
			"modules": compilerInterface.NewMap(map[string]*compilerInterface.Value{
				"datetime": funcMapAsValue(datetimeFunctions),
			}),

			// https://docs.getdbt.com/reference/dbt-jinja-functions/project_name
			"project_name": compilerInterface.NewString("PROJECT NAME"), // FIXME

			// https://docs.getdbt.com/reference/dbt-jinja-functions/target
			"target": compilerInterface.NewMap(map[string]*compilerInterface.Value{
				"name":    compilerInterface.NewString("dev"),
				"schema":  compilerInterface.NewString("FIXME-DATASET"),
				"type":    compilerInterface.NewString("bigquery"),
				"threads": compilerInterface.NewNumber(float64(utils.NumberWorkers)),
				"project": compilerInterface.NewString("FIXME-PROJECT"),

				//"dataset": compilerInterface.NewString("FIXME-DATASET"),
			}),
		},
	}
}

func (g *GlobalContext) SetVariable(name string, value *compilerInterface.Value) {
	panic("Cannot set variable on parentContext context - read only during execution")
}

func (g *GlobalContext) GetVariable(name string) *compilerInterface.Value {
	// Check the built in functions first
	builtInFunction := builtInFunctions[name]

	// Then check if a macro has been defined
	macro, err := g.GetMacro(name)
	if err == nil && macro != nil {
		// If macro's rely on each other, they may not be compiled yet and they will seperately
		// so we can ignore the error
		builtInFunction = macro
	}

	// Then check the local variable map
	variable, found := g.constants[name]
	if !found {
		if builtInFunction != nil {
			return compilerInterface.NewFunction(builtInFunction)
		} else {
			return &compilerInterface.Value{IsUndefined: true}
		}
	} else {
		return variable
	}
}

func (g *GlobalContext) ErrorAt(part compilerInterface.AST, error string) error {
	panic("implement me")
}

func (g *GlobalContext) NilResultFor(part compilerInterface.AST) error {
	panic("implement me")
}

func (g *GlobalContext) PushState() compilerInterface.ExecutionContext {
	panic("implement me")
}

func (g *GlobalContext) GetMacro(name string) (compilerInterface.FunctionDef, error) {
	g.macroMutex.RLock()
	macro, found := g.macros[name]
	g.macroMutex.RUnlock()

	// Check if it's compiled and registered
	if !found {
		// Do we have a macro file which isn't compiled yet?
		if file := g.fileSystem.Macro(name); file != nil {
			// Compile it
			if err := ParseFile(file); err != nil {
				return nil, err
			}

			if _, err := CompileModel(file, g); err != nil {
				return nil, err
			}

			// Attempt to re-read the compiled macro
			g.macroMutex.RLock()
			macro, found = g.macros[name]
			g.macroMutex.RUnlock()

			// If it's still not found, then the macro is not registering it self with it's filename
			if !found {
				return nil, errors.New(fmt.Sprintf("The macro file %s is not registering a macro with the same name!", name))
			}
		} else {
			// No macro exists for this
			return nil, nil
		}
	}

	return func(ec compilerInterface.ExecutionContext, caller compilerInterface.AST, args compilerInterface.Arguments) (*compilerInterface.Value, error) {
		// Note: we override the ExecutionContext with that from the original macro file
		//       but keep the caller reference
		newEC := macro.ec.PushState()
		newEC.SetVariable("caller", ec.GetVariable("caller"))

		return macro.function(newEC, caller, args)
	}, nil
}

func (g *GlobalContext) RegisterMacro(name string, ec compilerInterface.ExecutionContext, function compilerInterface.FunctionDef) {
	g.macroMutex.Lock()
	defer g.macroMutex.Unlock()

	g.macros[name] = &macroDef{
		ec:       ec,
		function: function,
	}
}
