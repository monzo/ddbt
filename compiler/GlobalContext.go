package compiler

import (
	"errors"
	"fmt"
	"sync"

	"ddbt/compiler/dbtUtils"
	"ddbt/compilerInterface"
	"ddbt/config"
	"ddbt/fs"
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
	fileName string
}

var _ compilerInterface.ExecutionContext = &GlobalContext{}

func NewGlobalContext(cfg *config.Config, fileSystem *fs.FileSystem) *GlobalContext {
	return &GlobalContext{
		fileSystem: fileSystem,
		macros:     make(map[string]*macroDef),
		constants: map[string]*compilerInterface.Value{
			"adapter": funcMapAsValue(adapterFunctions),

			"dbt_utils": funcMapAsValue(map[string]compilerInterface.FunctionDef{
				"union_all_tables":  dbtUtils.UnionAllTables,
				"get_column_values": dbtUtils.GetColumnValues,
				"pivot":             dbtUtils.Pivot,
				"unpivot":           dbtUtils.Unpivot,
				"group_by":          dbtUtils.GroupBy,
			}),

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
			"project_name": compilerInterface.NewString(cfg.Name),

			// https://docs.getdbt.com/reference/dbt-jinja-functions/target
			"target": compilerInterface.NewMap(map[string]*compilerInterface.Value{
				"name":    compilerInterface.NewString(cfg.Target.Name),
				"schema":  compilerInterface.NewString(cfg.Target.DataSet),
				"type":    compilerInterface.NewString("bigquery"),
				"threads": compilerInterface.NewNumber(float64(cfg.Target.Threads)),
				"project": compilerInterface.NewString(cfg.Target.ProjectID),
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
	panic("ErrorAt not implemented for global context")
}

func (g *GlobalContext) NilResultFor(part compilerInterface.AST) error {
	panic("NilResultFor not implemented for global context")
}

func (g *GlobalContext) PushState() compilerInterface.ExecutionContext {
	panic("PushState not implemented for global context")
}

func (g *GlobalContext) CopyVariablesInto(_ compilerInterface.ExecutionContext) {
	// No-op
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

			if err := CompileModel(file, g, true); err != nil {
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
		if _, err := ec.RegisterUpstreamAndGetRef(macro.fileName, fs.MacroFile); err != nil {
			return nil, ec.ErrorAt(caller, err.Error())
		}

		newEC := ec.PushState()
		// Note we copy any varaibles defined within the macro's own file in to the context being executed here too
		macro.ec.CopyVariablesInto(newEC)
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
		fileName: ec.FileName(),
	}
}

func (g *GlobalContext) RegisterUpstreamAndGetRef(name string, fileType string) (*compilerInterface.Value, error) {
	panic("RegisterUpstreamAndGetRef not implemented for global context")
}

func (g *GlobalContext) FileName() string {
	panic("FileName not implemented for global context")
}

func (g *GlobalContext) GetTarget() (*config.Target, error) {
	panic("GetTarget not implemented for global context")
}

func (g *GlobalContext) MarkAsDynamicSQL() (*compilerInterface.Value, error) {
	panic("Mark as dynamic SQL not support on the global context")
}
