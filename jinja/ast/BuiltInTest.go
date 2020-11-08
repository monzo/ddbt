package ast

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"unicode"

	"ddbt/compilerInterface"
	"ddbt/jinja/lexer"
)

type BuiltInTest struct {
	position  lexer.Position
	inverted  bool
	condition AST
	checkType string
	argument  AST
}

var _ AST = &BuiltInTest{}

func NewBuiltInTest(token *lexer.Token, inverted bool, condition AST, checkType string, arg AST) *BuiltInTest {
	return &BuiltInTest{
		position:  token.Start,
		inverted:  inverted,
		condition: condition,
		checkType: checkType,
		argument:  arg,
	}
}

func (op *BuiltInTest) Position() lexer.Position {
	return op.position
}

func (op *BuiltInTest) Execute(ec compilerInterface.ExecutionContext) (*compilerInterface.Value, error) {
	value, err := op.condition.Execute(ec)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, ec.NilResultFor(op.condition)
	}
	value = value.Unwrap()

	// If we have an argument, resolve it
	var argument *compilerInterface.Value
	if op.argument != nil {
		argument, err = op.argument.Execute(ec)
		if err != nil {
			return nil, err
		}

		if argument == nil {
			return nil, ec.NilResultFor(op.argument)
		}

		argument = argument.Unwrap()
	}

	// Find the test function
	test, found := BuiltInTests[op.checkType]
	if !found {
		return nil, ec.ErrorAt(op, fmt.Sprintf("Unknown test type `%s`", op.checkType))
	}

	// Execute it
	result, err := test(value, argument)
	if err != nil {
		return nil, ec.ErrorAt(op, err.Error())
	}

	// Invert the result if needed
	if op.inverted {
		result = !result
	}

	return compilerInterface.NewBoolean(result), nil
}

func (op *BuiltInTest) String() string {
	var builder strings.Builder

	builder.WriteString(op.condition.String())
	builder.WriteString(" is ")

	if op.inverted {
		builder.WriteString("not ")
	}

	builder.WriteString(op.checkType)

	if op.argument != nil {
		builder.WriteRune('(')
		builder.WriteString(op.argument.String())
		builder.WriteRune(')')
	}

	return builder.String()
}

var BuiltInTests = map[string]func(v, arg *compilerInterface.Value) (bool, error){
	"boolean": func(v, arg *compilerInterface.Value) (bool, error) {
		return v.Type() == compilerInterface.BooleanValue, nil
	},
	"callable": func(v, arg *compilerInterface.Value) (bool, error) { return v.Function != nil, nil },
	"defined":  func(v, arg *compilerInterface.Value) (bool, error) { return !v.IsUndefined, nil },
	"divisibleby": func(v, arg *compilerInterface.Value) (bool, error) {
		if arg == nil {
			return false, errors.New("divisibleby requires an argument")
		}

		value, err := v.AsNumberValue()
		if err != nil {
			return false, err
		}

		divisor, err := arg.AsNumberValue()
		if err != nil {
			return false, err
		}

		return math.Mod(value, divisor) == 0, nil
	},
	"eq": func(v, arg *compilerInterface.Value) (bool, error) {
		if arg == nil {
			return false, errors.New("eq requires an argument")
		}

		return v.Equals(arg), nil
	},
	"even": func(v, arg *compilerInterface.Value) (bool, error) {
		value, err := v.AsNumberValue()
		if err != nil {
			return false, err
		}

		return math.Mod(value, 2) == 0, nil
	},
	"FALSE": func(v, arg *compilerInterface.Value) (bool, error) {
		return v.Type() == compilerInterface.BooleanValue && !v.BooleanValue, nil
	},
	"float": func(v, arg *compilerInterface.Value) (bool, error) {
		return v.Type() == compilerInterface.NumberVal && math.Mod(v.NumberValue, 1) != 0, nil
	},
	"ge": func(v, arg *compilerInterface.Value) (bool, error) {
		if arg == nil {
			return false, errors.New("ge requires an argument")
		}

		lhsNum, err := v.AsNumberValue()
		if err != nil {
			return false, err
		}

		rhsNum, err := arg.AsNumberValue()
		if err != nil {
			return false, err
		}

		return lhsNum >= rhsNum, nil
	},
	"gt": func(v, arg *compilerInterface.Value) (bool, error) {
		if arg == nil {
			return false, errors.New("gt requires an argument")
		}

		lhsNum, err := v.AsNumberValue()
		if err != nil {
			return false, err
		}

		rhsNum, err := arg.AsNumberValue()
		if err != nil {
			return false, err
		}

		return lhsNum > rhsNum, nil
	},
	"in": func(needle, haystack *compilerInterface.Value) (bool, error) {
		if haystack == nil {
			return false, errors.New("in requires an argument")
		}

		switch haystack.Type() {
		case compilerInterface.StringVal:
			// substring check
			needleStr := needle.AsStringValue()

			return strings.Contains(haystack.StringValue, needleStr), nil

		case compilerInterface.ListVal:
			// value check
			for _, item := range haystack.ListValue {
				if item.Equals(needle) {
					return true, nil
				}
			}

			return false, nil

		case compilerInterface.MapVal:
			// key check
			needleStr := needle.AsStringValue()

			_, found := haystack.MapValue[needleStr]
			return found, nil

		default:
			return false, errors.New(fmt.Sprintf("Unable to perform the `in` operation on a %s", haystack.Type()))
		}
	},
	"integer": func(v, arg *compilerInterface.Value) (bool, error) {
		return v.Type() == compilerInterface.NumberVal && math.Mod(v.NumberValue, 1) == 0, nil
	},
	"iterable": func(v, arg *compilerInterface.Value) (bool, error) {
		return v.Type() == compilerInterface.ListVal || v.Type() == compilerInterface.MapVal, nil
	},
	"le": func(v, arg *compilerInterface.Value) (bool, error) {
		if arg == nil {
			return false, errors.New("le requires an argument")
		}

		lhsNum, err := v.AsNumberValue()
		if err != nil {
			return false, err
		}

		rhsNum, err := arg.AsNumberValue()
		if err != nil {
			return false, err
		}

		return lhsNum <= rhsNum, nil
	},
	"lower": func(v, arg *compilerInterface.Value) (bool, error) {
		for _, r := range v.AsStringValue() {
			if !unicode.IsLower(r) && unicode.IsLetter(r) {
				return false, nil
			}
		}
		return true, nil
	},
	"lt": func(v, arg *compilerInterface.Value) (bool, error) {
		if arg == nil {
			return false, errors.New("lt requires an argument")
		}

		lhsNum, err := v.AsNumberValue()
		if err != nil {
			return false, err
		}

		rhsNum, err := arg.AsNumberValue()
		if err != nil {
			return false, err
		}

		return lhsNum < rhsNum, nil
	},
	"mapping": func(v, arg *compilerInterface.Value) (bool, error) {
		return v.Type() == compilerInterface.MapVal, nil
	},
	"ne": func(v, arg *compilerInterface.Value) (bool, error) {
		if arg == nil {
			return false, errors.New("ne requires an argument")
		}

		return !v.Equals(arg), nil
	},
	"None": func(v, arg *compilerInterface.Value) (bool, error) { return v.IsNull || v.IsUndefined, nil },
	"number": func(v, arg *compilerInterface.Value) (bool, error) {
		return v.Type() == compilerInterface.NumberVal, nil
	},
	"odd": func(v, arg *compilerInterface.Value) (bool, error) {
		value, err := v.AsNumberValue()
		if err != nil {
			return false, err
		}

		return math.Mod(value, 2) != 0, nil
	},
	"sameas": func(v, arg *compilerInterface.Value) (bool, error) {
		if arg == nil {
			return false, errors.New("sameas requires an argument")
		}

		return v == arg, nil
	},
	"sequence": func(v, arg *compilerInterface.Value) (bool, error) {
		return v.Type() == compilerInterface.ListVal || v.Type() == compilerInterface.MapVal, nil
	},
	"string": func(v, arg *compilerInterface.Value) (bool, error) {
		return v.Type() == compilerInterface.StringVal, nil
	},
	"TRUE": func(v, arg *compilerInterface.Value) (bool, error) {
		return v.Type() == compilerInterface.BooleanValue && v.BooleanValue, nil
	},
	"undefined": func(v, arg *compilerInterface.Value) (bool, error) { return v.IsUndefined, nil },
	"upper": func(v, arg *compilerInterface.Value) (bool, error) {
		for _, r := range v.AsStringValue() {
			if !unicode.IsUpper(r) && unicode.IsLetter(r) {
				return false, nil
			}
		}
		return true, nil
	},
}

func init() {
	var builtInAliases = map[string]string{
		"==":          "eq",
		"equalto":     "eq",
		">=":          "ge",
		">":           "gt",
		"greaterthan": "gt",
		"<=":          "le",
		"<":           "lt",
		"lessthan":    "lt",
		"!=":          "ne",
	}

	for alias, aliasTo := range builtInAliases {
		to, found := BuiltInTests[aliasTo]
		if !found {
			panic(fmt.Sprintf("unable to alias %s to %s. %s not found!", alias, aliasTo, aliasTo))
		}

		BuiltInTests[alias] = to
	}
}
