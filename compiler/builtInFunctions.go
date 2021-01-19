package compiler

import (
	"fmt"
	"os"
	"strings"

	"ddbt/compilerInterface"
	"ddbt/fs"
	"ddbt/utils"
)

var builtInFunctions = map[string]compilerInterface.FunctionDef{
	// As ordered and listed in https://docs.getdbt.com/reference/dbt-jinja-functions
	"adapter": nil, // Note this is defined by the global context

	"as_bool": func(ec compilerInterface.ExecutionContext, caller compilerInterface.AST, args compilerInterface.Arguments) (*compilerInterface.Value, error) {
		if err := expectArgs(ec, caller, "as_bool", 1, args); err != nil {
			return nil, err
		}

		return compilerInterface.NewBoolean(args[0].Value.TruthyValue()), nil
	},

	"as_native": notImplemented(),

	"as_number": func(ec compilerInterface.ExecutionContext, caller compilerInterface.AST, args compilerInterface.Arguments) (*compilerInterface.Value, error) {
		if err := expectArgs(ec, caller, "as_number", 1, args); err != nil {
			return nil, err
		}

		if num, err := args[0].Value.AsNumberValue(); err != nil {
			return nil, ec.ErrorAt(caller, fmt.Sprintf("%s", err))
		} else {
			return compilerInterface.NewNumber(num), nil
		}
	},

	"as_text": func(ec compilerInterface.ExecutionContext, caller compilerInterface.AST, args compilerInterface.Arguments) (*compilerInterface.Value, error) {
		if err := expectArgs(ec, caller, "as_text", 1, args); err != nil {
			return nil, err
		}

		return compilerInterface.NewString(args[0].Value.AsStringValue()), nil
	},

	"builtins": notImplemented(),

	"config": nil, // Note this is defined by the compiler when creating the original execution context

	"dbt_version": func(ec compilerInterface.ExecutionContext, caller compilerInterface.AST, args compilerInterface.Arguments) (*compilerInterface.Value, error) {
		return compilerInterface.NewString(utils.DdbtVersion), nil
	},

	"debug": notImplemented(),

	"doc": notImplemented(),

	"env_var": func(ec compilerInterface.ExecutionContext, caller compilerInterface.AST, args compilerInterface.Arguments) (*compilerInterface.Value, error) {
		values, err := requiredArgs(ec, caller, args, "env_var", compilerInterface.StringVal)
		if err != nil {
			return nil, err
		}

		value := os.Getenv(values[0].AsStringValue())
		if value == "" && len(args) > 1 {
			value = args[1].Value.AsStringValue()
		}

		return compilerInterface.NewString(value), nil
	},

	"exceptions": nil, // Note this is defined in the global context

	"execute": nil, // Note this is defined in the global context

	"flags": notImplemented(),

	"fromjson": notImplemented(),

	"fromyaml": notImplemented(),

	"graph": notImplemented(),

	"invocation_id": notImplemented(),

	"log": func(ec compilerInterface.ExecutionContext, caller compilerInterface.AST, args compilerInterface.Arguments) (*compilerInterface.Value, error) {
		var builder strings.Builder

		for _, arg := range args {
			builder.WriteString(arg.Value.AsStringValue())
		}

		fmt.Printf("%s @ %s:%d:%d\n\n", builder.String(), caller.Position().File, caller.Position().Row, caller.Position().Column)

		return compilerInterface.NewUndefined(), nil
	},

	"modules": nil, // Note this is defined in the global context

	"project_name": nil, // Note this is defined in the global context

	"ref": refFunction,

	"replace": func(ec compilerInterface.ExecutionContext, caller compilerInterface.AST, args compilerInterface.Arguments) (*compilerInterface.Value, error) {
		if len(args) != 3 {
			return nil, ec.ErrorAt(caller, fmt.Sprintf("replace requires 3 parameters, got %d", len(args)))
		}

		value := strings.Replace(
			args[0].Value.AsStringValue(),
			args[1].Value.AsStringValue(),
			args[2].Value.AsStringValue(),
			1,
		)
		return compilerInterface.NewString(value), nil
	},

	"return": func(ec compilerInterface.ExecutionContext, caller compilerInterface.AST, args compilerInterface.Arguments) (*compilerInterface.Value, error) {
		if len(args) != 1 {
			return nil, ec.ErrorAt(caller, fmt.Sprintf("return requires 1 parameter, got %d", len(args)))
		}

		return compilerInterface.NewReturnValue(args[0].Value), nil
	},

	"run_query": notImplemented(),

	"run_started_at": notImplemented(),

	"schema": notImplemented(),

	"source": notImplemented(),

	"statement": notImplemented(),

	"target": nil, // Note this is defined in the global context

	"this": nil, // Note this is defined by the compiler when creating the original execution context

	"tojson": notImplemented(),

	"toyaml": notImplemented(),

	"var": func(ec compilerInterface.ExecutionContext, caller compilerInterface.AST, args compilerInterface.Arguments) (*compilerInterface.Value, error) {
		if len(args) < 1 {
			return nil, ec.ErrorAt(caller, fmt.Sprintf("var requires at least 1 parameter, got %d", len(args)))
		}

		// FIXME: implement lookup to project level information

		if len(args) > 1 {
			return args[1].Value, nil
		} else {
			return compilerInterface.NewUndefined(), nil
		}
	},

	// Extra's not in their main list
	// https://docs.getdbt.com/docs/building-a-dbt-project/building-models/configuring-incremental-models/#filtering-rows-on-an-incremental-run
	"is_incremental": func(ec compilerInterface.ExecutionContext, caller compilerInterface.AST, args compilerInterface.Arguments) (*compilerInterface.Value, error) {
		return compilerInterface.NewBoolean(false), nil
	},

	// Jinja2 Filter functions
	"lower": func(ec compilerInterface.ExecutionContext, caller compilerInterface.AST, args compilerInterface.Arguments) (*compilerInterface.Value, error) {
		values, err := requiredArgs(ec, caller, args, "lower", compilerInterface.StringVal)
		if err != nil {
			return nil, err
		}

		return compilerInterface.NewString(strings.ToLower(values[0].AsStringValue())), nil
	},

	"default": func(ec compilerInterface.ExecutionContext, caller compilerInterface.AST, args compilerInterface.Arguments) (*compilerInterface.Value, error) {
		if len(args) != 2 {
			return nil, ec.ErrorAt(caller, fmt.Sprintf("default requires 2 argumnets, got %d", len(args)))
		}

		if args[0].Value.IsUndefined || args[0].Value.IsNull {
			return args[1].Value, nil
		} else {
			return args[0].Value, nil
		}
	},

	// Our specific functions
	"indirect_ref": refFunction,

	// DDBT Debugging function - Allows removing of macro's from models without completely removing them
	"noop": func(ec compilerInterface.ExecutionContext, caller compilerInterface.AST, args compilerInterface.Arguments) (*compilerInterface.Value, error) {
		return compilerInterface.NewUndefined(), nil
	},
}

// The date time module from https://docs.getdbt.com/reference/dbt-jinja-functions/modules
var datetimeFunctions = map[string]compilerInterface.FunctionDef{
	// FIXME: we may need to implement this better
	"time": func(ec compilerInterface.ExecutionContext, caller compilerInterface.AST, args compilerInterface.Arguments) (*compilerInterface.Value, error) {
		values, err := requiredArgs(
			ec, caller, args, "modules.datetime.time",
			compilerInterface.NumberVal, compilerInterface.NumberVal, compilerInterface.NumberVal,
		)
		if err != nil {
			return nil, err
		}

		return compilerInterface.NewString(fmt.Sprintf("%2.f:%2.f:%2.f", values[0].NumberValue, values[0].NumberValue, values[0].NumberValue)), nil
	},

	"timedelta": func(ec compilerInterface.ExecutionContext, caller compilerInterface.AST, args compilerInterface.Arguments) (*compilerInterface.Value, error) {
		values, err := requiredArgs(
			ec, caller, args, "modules.datetime.timedelta",
			compilerInterface.NumberVal,
		)
		if err != nil {
			return nil, err
		}

		return compilerInterface.NewString(args[0].Name + "=" + values[0].AsStringValue()), nil
	},
}

//https://docs.getdbt.com/reference/dbt-jinja-functions/adapter
var adapterFunctions = map[string]compilerInterface.FunctionDef{
	"dispatch":                   noopMethod(),
	"get_missing_columns":        noopMethod(),
	"expand_target_column_types": noopMethod(),
	"get_relation":               noopMethod(),
	"get_columns_in_relation":    noopMethod(),
	"create_schema":              noopMethod(),
	"drop_schema":                noopMethod(),
	"drop_relation":              noopMethod(),
	"rename_relation":            noopMethod(),
	"get_columns_in_table":       noopMethod(),
	"already_exists":             noopMethod(),
	"adapter_macro":              noopMethod(),

	// Note listed on their site
	"check_schema_exists": noopMethod(),
}

func notImplemented() compilerInterface.FunctionDef {
	return func(ec compilerInterface.ExecutionContext, caller compilerInterface.AST, args compilerInterface.Arguments) (*compilerInterface.Value, error) {
		return nil, ec.ErrorAt(caller, "not implemented yet")
	}
}

func noopMethod() compilerInterface.FunctionDef {
	return func(ec compilerInterface.ExecutionContext, caller compilerInterface.AST, args compilerInterface.Arguments) (*compilerInterface.Value, error) {
		return compilerInterface.NewUndefined(), nil
	}
}

func refFunction(ec compilerInterface.ExecutionContext, caller compilerInterface.AST, args compilerInterface.Arguments) (*compilerInterface.Value, error) {
	// This is dynamic because a reference _might_ change due to the tags in the referenced model when they are compiled
	// thus we'll need to recompile _this_ model before executing it to pickup those changes
	if isOnlyCompilingSQL(ec) {
		if _, err := ec.MarkAsDynamicSQL(); err != nil {
			return nil, err
		}
	}

	values, err := requiredArgs(ec, caller, args, "ref", compilerInterface.StringVal)
	if err != nil {
		return nil, err
	}

	modelName := values[0].AsStringValue()

	return ec.RegisterUpstreamAndGetRef(modelName, fs.ModelFile)
}

type funcMap = map[string]compilerInterface.FunctionDef

func funcMapAsValue(in funcMap) *compilerInterface.Value {
	rtn := make(map[string]*compilerInterface.Value)

	for key, f := range in {
		rtn[key] = compilerInterface.NewFunction(f)
	}

	return compilerInterface.NewMap(rtn)
}

func requiredArgs(ec compilerInterface.ExecutionContext, caller compilerInterface.AST, args compilerInterface.Arguments, funcName string, types ...compilerInterface.ValueType) ([]*compilerInterface.Value, error) {
	if len(args) < len(types) {
		return nil, ec.ErrorAt(caller, fmt.Sprintf("%s expected %d arguments, got %d", funcName, len(types), len(args)))
	}

	values := make([]*compilerInterface.Value, 0, len(types))

	for i, expectedType := range types {
		value := args[i].Value
		if value == nil {
			return nil, ec.ErrorAt(caller, fmt.Sprintf("%s's argument %d was nil", funcName, i+1))
		}

		if value.Type() != expectedType {
			return nil, ec.ErrorAt(caller, fmt.Sprintf("%s's argument %d needs to be a %s, got %s", funcName, i+1, expectedType, value.Type()))
		}

		values = append(values, value)
	}

	return values, nil
}

func expectArgs(ec compilerInterface.ExecutionContext, caller compilerInterface.AST, funcName string, expect int, got compilerInterface.Arguments) error {
	if expect != len(got) {
		return ec.ErrorAt(caller, fmt.Sprintf("%s expected %d arguments, got %d", funcName, expect, len(got)))
	}
	return nil
}
