package dbtUtils

import (
	"context"
	"ddbt/bigquery"
	"ddbt/compilerInterface"
	"fmt"
	"strconv"
	"strings"
)

// GetColumnValues is a fallback GetColumnValuesWithContext
// with a background context.
func GetColumnValues(ec compilerInterface.ExecutionContext, caller compilerInterface.AST, arguments compilerInterface.Arguments) (*compilerInterface.Value, error) {
	return GetColumnValuesWithContext(context.Background(), ec, caller, arguments)
}

func GetColumnValuesWithContext(ctx context.Context, ec compilerInterface.ExecutionContext, caller compilerInterface.AST, arguments compilerInterface.Arguments) (*compilerInterface.Value, error) {
	if isOnlyCompilingSQL(ec) {
		return ec.MarkAsDynamicSQL()
	}

	args, err := getArgs(arguments, param("table"), param("column"), param("max_records"))
	if err != nil {
		return nil, ec.ErrorAt(caller, fmt.Sprintf("%s", err))
	}

	// Build a query to execute
	query := fmt.Sprintf(
		"SELECT %s as value FROM %s GROUP BY 1 ORDER BY COUNT(*) DESC",
		args[1].AsStringValue(),
		args[0].AsStringValue(),
	)

	if !args[2].IsUndefined {
		num, err := args[2].AsNumberValue()
		if err != nil {
			return nil, ec.ErrorAt(caller, fmt.Sprintf("%s", err))
		}

		query += " LIMIT " + strconv.Itoa(int(num))
	}

	target, err := ec.GetTarget()
	if err != nil {
		return nil, ec.ErrorAt(caller, fmt.Sprintf("%s", err))
	}

	rows, _, err := bigquery.GetRows(ctx, query, target)
	if err != nil {
		return nil, ec.ErrorAt(caller, fmt.Sprintf("get_column_values query returned an error: %s", err))
	}

	result := make([]*compilerInterface.Value, len(rows))

	for i, row := range rows {
		r, err := compilerInterface.NewValueFromInterface(row[0])
		if err != nil {
			return nil, ec.ErrorAt(caller, fmt.Sprintf("get_column_values was unable to parse a value: %s", err))
		}

		result[i] = r
	}

	return compilerInterface.NewList(result), nil
}

func Unpivot(ec compilerInterface.ExecutionContext, caller compilerInterface.AST, arguments compilerInterface.Arguments) (*compilerInterface.Value, error) {
	if isOnlyCompilingSQL(ec) {
		return ec.MarkAsDynamicSQL()
	}

	args, err := getArgs(arguments,
		paramWithDefault("table", compilerInterface.NewString("")),
		paramWithDefault("cast_to", compilerInterface.NewString("varchar")),
		paramWithDefault("exclude", compilerInterface.NewList(make([]*compilerInterface.Value, 0))),
		paramWithDefault("remove", compilerInterface.NewList(make([]*compilerInterface.Value, 0))),
		paramWithDefault("field_name", compilerInterface.NewString("field_name")),
		paramWithDefault("value_name", compilerInterface.NewString("value_name")),
	)
	if err != nil {
		return nil, ec.ErrorAt(caller, fmt.Sprintf("%s", err))
	}

	table := args[0].AsStringValue()
	castTo := args[1].AsStringValue()
	exclude := args[2].ListValue
	remove := listToSet(args[3].ListValue)
	fieldName := args[4].AsStringValue()
	valueName := args[5].AsStringValue()

	excludeSet := listToSet(exclude)

	target, err := ec.GetTarget()
	if err != nil {
		return nil, ec.ErrorAt(caller, fmt.Sprintf("%s", err))
	}

	columns, err := bigquery.GetColumnsFromTable(table, target)
	if err != nil {
		return nil, ec.ErrorAt(caller, fmt.Sprintf("Unable to get the columns for %s: %s", table, err))
	}

	var builder strings.Builder

	includeColumns := make([]string, 0)
	for _, col := range columns {
		lowered := strings.ToLower(col.Name)

		if _, found := excludeSet[lowered]; found {
			continue
		}

		if _, found := remove[lowered]; found {
			continue
		}

		includeColumns = append(includeColumns, col.Name)
	}

	for i, col := range includeColumns {
		if i > 0 {
			builder.WriteString("\n UNION ALL \n")
		}

		builder.WriteString("SELECT \n\t")

		for _, excluded := range exclude {
			builder.WriteString(excluded.AsStringValue())
			builder.WriteString(",\n\t")
		}

		//       cast('{{ col.column }}' as {{ dbt_utils.type_string() }}) as {{ field_name }},
		builder.WriteString("CAST('")
		builder.WriteString(col)
		builder.WriteString("' AS STRING) AS ")
		builder.WriteString(fieldName)
		builder.WriteString(",\n\t")

		//       cast({{ col.column }} as {{ cast_to }}) as {{ value_name }}
		builder.WriteString("CAST(")
		builder.WriteString(col)
		builder.WriteString(" AS ")
		builder.WriteString(castTo)
		builder.WriteString(") AS ")
		builder.WriteString(valueName)
		builder.WriteString("\n\t")

		builder.WriteString("FROM ")
		builder.WriteString(table)
	}

	return compilerInterface.NewString(builder.String()), nil
}
