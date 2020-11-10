package compiler

import "ddbt/fs"

// All our built in Macros
const builtInMacros = `
{# This test checks that the value in column_name is always unique #}
{% macro test_unique(model, column_name) %}
WITH test_data AS (
	SELECT
	{{ column_name }} AS value,
	COUNT({{ column_name }}) AS count
	
	FROM {{ model }}
	
	GROUP BY {{ column_name }} 
	
	HAVING COUNT({{ column_name }}) > 1
)

SELECT COUNT(*) as num_errors FROM test_data
{% endmacro %}


{# This test that the value is never null in column_name #}
{% macro test_not_null(model, column_name) %}
WITH test_data AS (
	SELECT
	{{ column_name }} AS value
	
	FROM {{ model }}
	
	WHERE {{ column_name }} IS NULL
)

SELECT COUNT(*) as num_errors FROM test_data
{% endmacro %}


{# This test checks that the value in column_name is always one of the #} 
{% macro test_accepted_values(model, column_name, values) %}
WITH test_data AS (
	SELECT
	{{ column_name }} AS value
	
	FROM {{ model }}

	WHERE {{ column_name }} NOT IN (
		{% for value in values -%}
			{% if value is string and kwargs.get('quote', true) %}'{{ value }}'{% else %}{{value}}{% endif %}
			{%- if not loop.last %}, {% endif %} 
		{%- endfor %}
	)
)

SELECT COUNT(*) as num_errors FROM test_data
{% endmacro %}

{% macro test_relationships(model, column_name, to, field) %}
WITH test_data AS (
	SELECT
	{{ column_name }} AS value

	FROM {{ model }} AS src

	LEFT JOIN {{ to }} AS dest
	ON dest.{{ field }} = src.{{ column_name }}

	WHERE dest.{{ field }} IS NULL AND src.{{ column_name }} IS NOT NULL
)

SELECT COUNT(*) as num_errors FROM test_data
{% endmacro %}
`

// Adds and compiles in built in macros
func addBuiltInMacros(fileSystem *fs.FileSystem) error {
	file, err := fileSystem.AddMacroWithContents("built-in-macros", builtInMacros)
	if err != nil {
		return err
	}

	if err := ParseFile(file); err != nil {
		return err
	}

	return nil
}
