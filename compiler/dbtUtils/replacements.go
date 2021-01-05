package dbtUtils

import (
	"fmt"
	"strconv"
	"strings"

	"ddbt/bigquery"
	"ddbt/compilerInterface"
)

func UnionAllTables(ec compilerInterface.ExecutionContext, caller compilerInterface.AST, arguments compilerInterface.Arguments) (*compilerInterface.Value, error) {
	args, err := getArgs(arguments, param("tables"), param("column_names"))
	if err != nil {
		return nil, ec.ErrorAt(caller, fmt.Sprintf("%s", err))
	}

	if args[0].Type() != compilerInterface.ListVal || args[1].Type() != compilerInterface.ListVal {
		return nil, ec.ErrorAt(caller, "expected arguments to union all tables to be two lists")
	}

	var builder strings.Builder
	builder.WriteRune('(')
	for i, table := range args[0].ListValue {
		if i > 0 {
			builder.WriteString(" UNION ALL \n")
		}

		builder.WriteString("\n(SELECT ")

		for j, column := range args[1].ListValue {
			if j > 0 {
				builder.WriteString(", ")
			}

			builder.WriteString(column.AsStringValue())
		}

		builder.WriteString(" FROM ")
		builder.WriteString(table.AsStringValue())
		builder.WriteRune(')')
	}
	builder.WriteRune(')')
	return compilerInterface.NewString(builder.String()), nil
}

func GroupBy(ec compilerInterface.ExecutionContext, caller compilerInterface.AST, arguments compilerInterface.Arguments) (*compilerInterface.Value, error) {
	args, err := getArgs(arguments, paramWithDefault("n", compilerInterface.NewNumber(0)))
	if err != nil {
		return nil, ec.ErrorAt(caller, fmt.Sprintf("%s", err))
	}

	var builder strings.Builder
	builder.WriteString(" GROUP BY ")

	max := int(args[0].NumberValue) + 1

	for i := 1; i < max; i++ {
		if i > 1 {
			builder.WriteString(", ")
		}

		builder.WriteString(strconv.Itoa(i))
	}
	builder.WriteRune(' ')

	return compilerInterface.NewString(builder.String()), nil
}

func Pivot(ec compilerInterface.ExecutionContext, caller compilerInterface.AST, arguments compilerInterface.Arguments) (*compilerInterface.Value, error) {
	args, err := getArgs(arguments,
		paramWithDefault("column", compilerInterface.NewString("")),
		paramWithDefault("values", compilerInterface.NewList(make([]*compilerInterface.Value, 0))),
		paramWithDefault("alias", compilerInterface.NewBoolean(true)),
		paramWithDefault("agg", compilerInterface.NewString("sum")),
		paramWithDefault("cmp", compilerInterface.NewString("=")),
		paramWithDefault("prefix", compilerInterface.NewString("")),
		paramWithDefault("suffix", compilerInterface.NewString("")),
		param("then_value"),
		param("else_value"),
		paramWithDefault("quote_identifiers", compilerInterface.NewBoolean(true)),
	)
	if err != nil {
		return nil, ec.ErrorAt(caller, fmt.Sprintf("%s", err))
	}

	var builder strings.Builder

	column, values, alias, agg, cmp, prefix, suffix, then_value, else_value, quote_identifiers :=
		args[0].AsStringValue(), args[1].ListValue, args[2].BooleanValue, args[3].AsStringValue(), args[4].AsStringValue(), args[5].AsStringValue(), args[6].AsStringValue(), args[7], args[8], args[9].BooleanValue

	if then_value.IsUndefined {
		then_value = compilerInterface.NewNumber(1)
	}
	if else_value.IsUndefined {
		else_value = compilerInterface.NewNumber(0)
	}

	for i, value := range values {
		if i > 0 {
			builder.WriteRune(',')
		}

		// {{ agg }}( CASE WHEN {{ column }} {{ cmp }} '{{ v }}' THEN {{ then_value }} ELSE {{ else_value }} END )
		builder.WriteString(agg)
		builder.WriteString("(\nCASE WHEN ")
		builder.WriteString(column)
		builder.WriteRune(' ')
		builder.WriteString(cmp)
		builder.WriteRune('\'')
		builder.WriteString(value.AsStringValue())
		builder.WriteString("' THEN ")
		builder.WriteString(then_value.AsStringValue())
		builder.WriteString(" ELSE ")
		builder.WriteString(else_value.AsStringValue())
		builder.WriteString(" END)")

		if alias {
			builder.WriteString(" AS ")

			str := fmt.Sprintf("%s%s%s", prefix, value.AsStringValue(), suffix)
			if quote_identifiers {
				str = bigquery.Quote(str)
			}

			builder.WriteString(str)
		}

	}

	return compilerInterface.NewString(builder.String()), nil
}

func listToSet(list []*compilerInterface.Value) map[string]struct{} {
	set := make(map[string]struct{})

	for _, value := range list {
		set[strings.ToLower(value.AsStringValue())] = struct{}{}
	}

	return set
}
