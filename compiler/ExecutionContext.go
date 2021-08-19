package compiler

import (
	"fmt"
	"reflect"
	"sync"

	"ddbt/compilerInterface"
	"ddbt/config"
	"ddbt/fs"
)

type ExecutionContext struct {
	file           *fs.File
	fileSystem     *fs.FileSystem
	varaiblesMutex sync.RWMutex
	variables      map[string]*compilerInterface.Value
	states         []map[string]*compilerInterface.Value //nolint:golint,unused,structcheck
	isExecuting    bool

	globalContext *GlobalContext
	parentContext compilerInterface.ExecutionContext
}

// Ensure our execution context matches the interface in the AST package
var _ compilerInterface.ExecutionContext = &ExecutionContext{}

func NewExecutionContext(file *fs.File, fileSystem *fs.FileSystem, isExecuting bool, globalContext *GlobalContext, parent compilerInterface.ExecutionContext) *ExecutionContext {
	return &ExecutionContext{
		file:          file,
		fileSystem:    fileSystem,
		variables:     make(map[string]*compilerInterface.Value),
		isExecuting:   isExecuting,
		globalContext: globalContext,
		parentContext: parent,
	}
}

func (e *ExecutionContext) SetVariable(name string, value *compilerInterface.Value) {
	e.varaiblesMutex.Lock()
	e.variables[name] = value
	e.varaiblesMutex.Unlock()
}

func (e *ExecutionContext) GetVariable(name string) *compilerInterface.Value {
	e.varaiblesMutex.RLock()
	// Then check the local variable map
	variable, found := e.variables[name]
	e.varaiblesMutex.RUnlock()

	if !found {
		return e.parentContext.GetVariable(name)
	} else {
		return variable
	}
}

func (e *ExecutionContext) RegisterMacro(name string, ec compilerInterface.ExecutionContext, function compilerInterface.FunctionDef) {
	e.parentContext.RegisterMacro(name, ec, function)
}

func (e *ExecutionContext) ErrorAt(part compilerInterface.AST, error string) error {
	if part == nil {
		return fmt.Errorf("%s @ unknown", error)
	} else {
		pos := part.Position()
		return fmt.Errorf("%s @ %s:%d:%d", error, pos.File, pos.Row, pos.Column)
	}
}

func (e *ExecutionContext) NilResultFor(part compilerInterface.AST) error {
	return e.ErrorAt(part, fmt.Sprintf("%v returned a nil result after execution", reflect.TypeOf(part)))
}

func (e *ExecutionContext) PushState() compilerInterface.ExecutionContext {
	return NewExecutionContext(e.file, e.fileSystem, e.isExecuting, e.globalContext, e)
}

func (e *ExecutionContext) CopyVariablesInto(ec compilerInterface.ExecutionContext) {
	e.parentContext.CopyVariablesInto(ec)

	e.varaiblesMutex.RLock()
	defer e.varaiblesMutex.RUnlock()

	for key, value := range e.variables {
		ec.SetVariable(key, value)
	}
}

func (e *ExecutionContext) RegisterUpstreamAndGetRef(modelName string, fileType string) (*compilerInterface.Value, error) {
	var upstream *fs.File

	switch fileType {
	case fs.ModelFile:
		upstream = e.fileSystem.Model(modelName)

	case fs.MacroFile:
		upstream = e.fileSystem.Macro(modelName)

		if upstream == nil {
			// For tests
			upstream = e.fileSystem.Model(modelName)
		}

	default:
		return nil, fmt.Errorf("unknown file type: %s", fileType)
	}

	if upstream == nil {
		return nil, fmt.Errorf("Unable to find model `%s`", modelName)
	}

	e.file.RecordDependencyOn(upstream)

	target, err := upstream.GetTarget()
	if err != nil {
		return nil, err
	}

	switch upstream.GetMaterialization() {
	case "table", "incremental", "project_sharded_table", "view":
		//ToDo: views are being treated as tables until they are properly implemented

		// If "--upstream=target" has been provided and this model is not in the DAG, then we read from the upstream
		// target, rather than the target defined in "--target=target"
		if target.ReadUpstream != nil && !upstream.IsInDAG() {
			return compilerInterface.NewString(
				"`" + target.ReadUpstream.ProjectID + "`.`" + target.ReadUpstream.DataSet + "`.`" + modelName + "`",
			), nil
		} else {
			return compilerInterface.NewString(
				"`" + target.ProjectID + "`.`" + target.DataSet + "`.`" + modelName + "`",
			), nil
		}

	case "ephemeral":
		err := CompileModel(upstream, e.globalContext, e.isExecuting)
		if err != nil {
			return nil, err
		}

		cteName := fmt.Sprintf("__dbt__CTE__%s", upstream.Name)

		e.file.Mutex.Lock()
		e.file.EphemeralCTES[cteName] = upstream
		e.file.Mutex.Unlock()

		return compilerInterface.NewString(cteName), nil

	default:
		return nil, fmt.Errorf("unknown materialized config '%s' in model '%s'", upstream.GetMaterialization(), upstream.Name)
	}
}

func (e *ExecutionContext) FileName() string {
	return e.file.Name
}

func (e *ExecutionContext) GetTarget() (*config.Target, error) {
	return e.file.GetTarget()
}

func (e *ExecutionContext) MarkAsDynamicSQL() (*compilerInterface.Value, error) {
	e.file.MaskAsDynamicSQL()
	return compilerInterface.NewUndefined(), nil
}
