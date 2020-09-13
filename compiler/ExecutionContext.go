package compiler

import (
	"errors"
	"fmt"
	"reflect"

	"ddbt/compilerInterface"
)

type ExecutionContext struct {
	variables map[string]*compilerInterface.Value
	states    []map[string]*compilerInterface.Value
}

// Ensure our execution context matches the interface in the AST package
var _ compilerInterface.ExecutionContext = &ExecutionContext{}

func NewExecutionContext() *ExecutionContext {
	return &ExecutionContext{
		variables: make(map[string]*compilerInterface.Value),
	}
}

func (e *ExecutionContext) SetVariable(name string, value *compilerInterface.Value) {
	e.variables[name] = value
}

func (e *ExecutionContext) GetVariable(name string) *compilerInterface.Value {
	variable, found := e.variables[name]
	if !found {
		return &compilerInterface.Value{IsUndefined: true}
	} else {
		return variable
	}
}

func (e *ExecutionContext) ErrorAt(part compilerInterface.AST, error string) error {
	if part == nil {
		return errors.New(fmt.Sprintf("%s @ unknown", error))
	} else {
		pos := part.Position()
		return errors.New(fmt.Sprintf("%s @ %d:%d", error, pos.Row, pos.Column))
	}
}

func (e *ExecutionContext) NilResultFor(part compilerInterface.AST) error {
	return e.ErrorAt(part, fmt.Sprintf("%v returned a nil result after execution", reflect.TypeOf(part)))
}

func (e *ExecutionContext) PushState() {
	current := e.variables
	e.states = append(e.states, current)

	// Copy the current array
	e.variables = make(map[string]*compilerInterface.Value)
	for key, value := range current {
		e.variables[key] = value
	}
}

func (e *ExecutionContext) PopState() {
	if len(e.states) == 0 {
		panic("tried to pop too many states!")
	}

	e.variables = e.states[0]
	e.states = e.states[1:]
}
