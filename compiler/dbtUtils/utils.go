package dbtUtils

import (
	"ddbt/compilerInterface"
	"fmt"
)

func param(name string) compilerInterface.Argument {
	return paramWithDefault(name, nil)
}

func paramWithDefault(name string, value *compilerInterface.Value) compilerInterface.Argument {
	return compilerInterface.Argument{
		Name:  name,
		Value: value,
	}
}

func getArgs(arguments compilerInterface.Arguments, params ...compilerInterface.Argument) ([]*compilerInterface.Value, error) {
	args := make([]*compilerInterface.Value, len(params))

	// quick lookup map
	namedArgs := make(map[string]*compilerInterface.Value)
	for _, arg := range arguments {
		if arg.Name != "" {
			namedArgs[arg.Name] = arg.Value
		}
	}

	stillOrdered := true

	// Process all the parameters, checking what args where provided
	for i, param := range params {
		if value, found := namedArgs[param.Name]; found {
			args[i] = value

			stillOrdered = len(arguments) > i && arguments[i].Name == param.Name
		} else if len(arguments) <= i || arguments[i].Name != "" {
			stillOrdered = false
			if param.Value != nil {
				args[i] = param.Value
			} else {
				args[i] = compilerInterface.NewUndefined()
			}
		} else if !stillOrdered {
			return nil, fmt.Errorf("Named arguments have been used out of order, please either used all named arguments or keep them in order. Unable to identify what %s should be.", param.Name)
		} else {
			args[i] = arguments[i].Value
		}

		// Remove a return wrapper
		args[i] = args[i].Unwrap()

		// Check types
		if param.Value != nil && !args[i].IsUndefined {
			if param.Value.Type() != args[i].Type() {
				return nil, fmt.Errorf("Paramter %s should be a %s got a %s", param.Name, param.Value.Type(), args[i].Type())
			}
		}
	}

	return args, nil
}

func isOnlyCompilingSQL(ec compilerInterface.ExecutionContext) bool {
	value := ec.GetVariable("execute")

	if value.Type() == compilerInterface.BooleanValue {
		return !value.BooleanValue
	} else {
		return true
	}
}
