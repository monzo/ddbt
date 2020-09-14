package ast

import (
	"fmt"
	"sort"
	"strings"

	"ddbt/compilerInterface"
	"ddbt/jinja/lexer"
)

// A block which represents a simple
type ForLoop struct {
	position     lexer.Position
	keyItrName   string
	valueItrName string
	list         *Variable
	body         *Body
}

type ForLoopParameter struct {
	name         string
	defaultValue *lexer.Token
}

var _ AST = &ForLoop{}

func NewForLoop(valueItrToken *lexer.Token, keyItr string, list *Variable) *ForLoop {
	return &ForLoop{
		position:     valueItrToken.Start,
		keyItrName:   keyItr,
		valueItrName: valueItrToken.Value,
		list:         list,
		body:         NewBody(valueItrToken),
	}
}

func (fl *ForLoop) Position() lexer.Position {
	return fl.position
}

func (fl *ForLoop) Execute(ec compilerInterface.ExecutionContext) (*compilerInterface.Value, error) {
	// Get the list
	list, err := fl.list.Execute(ec)
	if err != nil {
		return nil, err
	}
	if list == nil {
		return nil, ec.NilResultFor(fl.list)
	}

	if list.ValueType == compilerInterface.ReturnVal {
		list = list.ReturnValue
	}

	switch list.Type() {
	case compilerInterface.ListVal:
		return fl.executeForList(list.ListValue, ec)

	case compilerInterface.MapVal:
		return fl.executeForMap(list.MapValue, ec)

	default:
		return nil, ec.ErrorAt(fl, fmt.Sprintf("unable to run for each over %s", list.Type()))
	}
}

func (fl *ForLoop) executeForList(list []*compilerInterface.Value, parentEC compilerInterface.ExecutionContext) (*compilerInterface.Value, error) {
	var builder strings.Builder

	for index, value := range list {
		ec := parentEC.PushState()

		// Set the loop variables
		ec.SetVariable("loop", &compilerInterface.Value{
			MapValue: map[string]*compilerInterface.Value{
				"index": compilerInterface.NewNumber(float64(index + 1)), // Python loops start at 1!!!
				"last":  compilerInterface.NewBoolean(index == (len(list) - 1)),
			},
		})
		if fl.keyItrName != "" {
			ec.SetVariable(fl.keyItrName, compilerInterface.NewNumber(float64(index)))
		}
		ec.SetVariable(fl.valueItrName, value)

		result, err := fl.body.Execute(ec)
		if err != nil {
			return nil, err
		}

		if err := writeValue(ec, fl.body, &builder, result); err != nil {
			return nil, err
		}
	}

	return &compilerInterface.Value{StringValue: builder.String()}, nil
}

func (fl *ForLoop) executeForMap(list map[string]*compilerInterface.Value, parentEC compilerInterface.ExecutionContext) (*compilerInterface.Value, error) {
	var builder strings.Builder

	num := 0

	// Sort keys so this loop excutes stably (i.e. the order doesn't change each time)
	keys := make([]string, 0, len(list))
	for key := range list {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for index, key := range keys {
		ec := parentEC.PushState()

		// Set the loop variables
		ec.SetVariable("loop", &compilerInterface.Value{
			MapValue: map[string]*compilerInterface.Value{
				"last": compilerInterface.NewBoolean(index == (len(list) - 1)),
			},
		})

		if fl.keyItrName != "" {
			ec.SetVariable(fl.keyItrName, compilerInterface.NewString(key))
		}
		ec.SetVariable(fl.valueItrName, list[key])

		result, err := fl.body.Execute(ec)
		if err != nil {
			return nil, err
		}

		if err := writeValue(ec, fl.body, &builder, result); err != nil {
			return nil, err
		}

		num++
	}

	return &compilerInterface.Value{StringValue: builder.String()}, nil
}

func (fl *ForLoop) String() string {
	if fl.keyItrName != "" {
		return fmt.Sprintf("\n{%% for %s, %s in %s %%}%s{%% endfor %%}", fl.keyItrName, fl.valueItrName, fl.list.String(), fl.body.String())
	} else {
		return fmt.Sprintf("\n{%% for %s in %s %%}%s{%% endfor %%}", fl.valueItrName, fl.list.String(), fl.body.String())
	}
}

func (fl *ForLoop) AppendBody(node AST) {
	fl.body.Append(node)
}
