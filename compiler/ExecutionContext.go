package compiler

import (
	"errors"
	"fmt"
	"reflect"
	"sync"

	"ddbt/compilerInterface"
)

type ExecutionContext struct {
	varaiblesMutex sync.RWMutex
	variables      map[string]*compilerInterface.Value
	states         []map[string]*compilerInterface.Value
	parentContext  compilerInterface.ExecutionContext
}

// Ensure our execution context matches the interface in the AST package
var _ compilerInterface.ExecutionContext = &ExecutionContext{}

func NewExecutionContext(parent compilerInterface.ExecutionContext) *ExecutionContext {
	return &ExecutionContext{
		variables:     make(map[string]*compilerInterface.Value),
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
		return errors.New(fmt.Sprintf("%s @ unknown", error))
	} else {
		pos := part.Position()
		return errors.New(fmt.Sprintf("%s @ %s:%d:%d", error, pos.File, pos.Row, pos.Column))
	}
}

func (e *ExecutionContext) NilResultFor(part compilerInterface.AST) error {
	return e.ErrorAt(part, fmt.Sprintf("%v returned a nil result after execution", reflect.TypeOf(part)))
}

func (e *ExecutionContext) PushState() compilerInterface.ExecutionContext {
	return NewExecutionContext(e)
}
