package schemaTestMacros

import "fmt"

func Test_unique_macro(project string, dataset string, model string, column_name string) (string, string) {
	return fmt.Sprintf(`select count(*)
	from (
		select
			%s
		from %s.%s.%s
		where %s is not null
		group by %s
		having count(*) > 1
	) validation_errors
	`, column_name, project, dataset, model, column_name, column_name), "unique"
}
