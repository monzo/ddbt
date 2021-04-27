package schemaTestMacros

func Test_not_null_macro() string {
	return `select count(*) 
	from {{ model }}
	where {{ column_name }} is null
	`
}
