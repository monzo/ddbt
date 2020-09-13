package tests

import (
	"testing"
)

func TestNoJinjaTemplating(t *testing.T) {
	const raw = "SELECT * FROM BLAH"
	assertCompileOutput(t, raw, raw)
}

func TestCommentBlocks(t *testing.T) {
	const raw = "SELECT * FROM BLAH"
	assertCompileOutput(t, raw, "{# Test Comment #}"+raw)
	assertCompileOutput(t, raw, "{# Test Comment \n\t#}"+raw)
	assertCompileOutput(t, raw, raw+"{# Test Comment #}")
	assertCompileOutput(t, raw, raw+"{#\n\t Test Comment #}")
	assertCompileOutput(t, raw, "SELECT {# test comment#}* FROM BLAH")
	assertCompileOutput(t, raw, "SELECT {# test \n\n\ttest\n\ncomment#}* FROM BLAH")
}

func TestBasicVariables(t *testing.T) {
	assertCompileOutput(t, "BLAH", "{{ table_name }}")
	assertCompileOutput(t, "1", "{{ number_value }}")
	assertCompileOutput(t, "2", "{{ str_number_value }}")

	const raw = "SELECT * FROM BLAH"
	assertCompileOutput(t, raw, "SELECT * FROM {{ table_name }}")
	assertCompileOutput(t, raw, "SELECT * FROM {{table_name}}")
	assertCompileOutput(t, raw, "SELECT * FROM {{ table_name}}")
	assertCompileOutput(t, raw, "SELECT * FROM {{table_name }}")
}

func TestListVariables(t *testing.T) {
	assertCompileOutput(t, "first option is string", "{{ list_object[0] }}")
	assertCompileOutput(t, "second option a string too!", "{{ list_object[1] }}")
	assertCompileOutput(t, "third", "{{ list_object[2] }}")
}

func TestMapVariables(t *testing.T) {
	assertCompileOutput(t, "test", "{{ map_object['string'] }}")
	assertCompileOutput(t, "42", "{{ map_object['key'] }}")
	assertCompileOutput(t, "test", "{{ map_object.string }}")
	assertCompileOutput(t, "42", "{{ map_object.key }}")
}

func TestComplexVariableCombination(t *testing.T) {
	assertCompileOutput(t, "3", "{{ map_object.nested.number }}")
	assertCompileOutput(t, "thingy", "{{ list_object[3].blah[0] }}")
	assertCompileOutput(t, "thingy", "{{ list_object[map_object.nested.number].blah[0] }}")
	assertCompileOutput(t, "thingy", "{{ list_object[map_object['nested']['number']].blah[0] }}")
	assertCompileOutput(t, "thingy", "{{ list_object[list_object[4][1]].blah[0] }}")
}

func TestSettingVariable(t *testing.T) {
	assertCompileOutput(
		t,
		"BEFORE: \n\nAFTER: 21",
		`BEFORE: {{ a_test_variable }}
{% set a_test_variable = 21 %}
AFTER: {{ a_test_variable }}`,
	)
}

func TestExpressionTrimSettings(t *testing.T) {
	assertCompileOutput(
		t,
		"BEFORE: \nAFTER: 21",
		`BEFORE: {{ a_test_variable }}
{% set a_test_variable = 21 -%}


AFTER: {{ a_test_variable }}`,
	)

	assertCompileOutput(
		t,
		"BEFORE: \nAFTER: 21",
		`BEFORE: {{ a_test_variable }}

{%- set a_test_variable = 21 %}
AFTER: {{ a_test_variable }}`,
	)

	assertCompileOutput(
		t,
		"BEFORE: AFTER: 21",
		`BEFORE: {{ a_test_variable }}

{%- set a_test_variable = 21 -%}




AFTER: {{ a_test_variable }}`,
	)
}

func TestListParsing(t *testing.T) {
	assertCompileOutput(t,
		"hello",
		`{% set list = [
	1,
	"two",
	"hello",
	false,
	null
] -%}

{{ list[2] }}`)
}

func TestMapParsing(t *testing.T) {
	assertCompileOutput(t,
		"barfoo",
		`{% set list = {
	'foo': 'bar',
	'bar': 'foo',
} -%}

{{ list['foo'] }}{{ list.bar }}`)
}

func TestNestedMapListParsing(t *testing.T) {
	assertCompileOutput(t,
		"bob",
		`{% set map = {
	'foo': 'bar',
	"list": [
		1,
		{ "name": "bob"},
	],
	'bar': 'foo',
} -%}

{{ map.list[1].name }}`)
}

func TestSimpleForLoopsWithLists(t *testing.T) {
	assertCompileOutput(t,
		"1 + 2 + 3 + 4 + ",
		`{% set list = [1, 2, 3, 4] -%}
{% for value in list %}{{ value }} + {% endfor %}`)

	assertCompileOutput(t,
		"0:94 - 1:32 - 2:11 - ",
		`{% set list = [94, 32, 11] -%}
{% for i, value in list %}{{i}}:{{ value }} - {% endfor %}`)
}

func TestNestedForLoopsWithLists(t *testing.T) {
	assertCompileOutput(t,
		"List 0: 0:z 1:y 2:x\nList 1: 0:a 1:b 2:c\n",
		`{%- set list = [
	[ "z", "y", "x"],
	[ "a", "b", "c"],
] -%}
{%- for i, list in list -%}
	List {{ i }}:
	{%- for j, letter in list%} {{ j }}:{{ letter }}{% endfor %}
{% endfor -%}
`)
}

func TestSimpleForLoopsWithMaps(t *testing.T) {
	assertCompileOutput(t,
		"23, Joe, ",
		`{% set list = { "name": "Joe", "age": 23 } -%}
{% for value in list %}{{ value }}, {% endfor %}`)

	assertCompileOutput(t,
		"age = 23\nname = Joe\n",
		`{% set list = { "name": "Joe", "age": 23 } -%}
{% for key, value in list %}{{ key }} = {{ value }}
{% endfor %}`)
}

func TestListItemsCall(t *testing.T) {
	assertCompileOutput(t,
		"1337",
		`{%- set list = [1, 3, 3, 7] -%}
{%- for number in list.items() -%}{{ number }}{% endfor -%}`)
}

func TestIfStatement(t *testing.T) {
	assertCompileOutput(t,
		"Passed",
		`{% if true %}Passed{% endif %}`)

	assertCompileOutput(t,
		"Passed",
		`{% if true %}Passed{% else %}Fail{% endif %}`)

	assertCompileOutput(t,
		"Passed",
		`{% if false %}Failed{% else %}Passed{% endif %}`)

	assertCompileOutput(t,
		"Passed",
		`{% if false %}Failed{% elif true %}Passed{% else %}Failed{% endif %}`)

	assertCompileOutput(t,
		"Passed",
		`{% if false %}Failed{% elif false %}Failed{% else %}Passed{% endif %}`)

	assertCompileOutput(t,
		"Passed",
		`{% if true %}Passed{% elif true %}Failed{% else %}Failed{% endif %}`)
}

func TestAndCondition(t *testing.T) {
	assertCompileOutput(t,
		"Passed",
		`{% if true and true %}Passed{% else %}Fail{% endif %}`)

	assertCompileOutput(t,
		"Passed",
		`{% if true and true and true and true %}Passed{% else %}Fail{% endif %}`)

	assertCompileOutput(t,
		"Passed",
		`{% if false and true %}Fail{% else %}Passed{% endif %}`)

	assertCompileOutput(t,
		"Passed",
		`{% if true and false %}Fail{% else %}Passed{% endif %}`)

	assertCompileOutput(t,
		"Passed",
		`{% if false and false %}Fail{% else %}Passed{% endif %}`)

	assertCompileOutput(t,
		"Passed",
		`{% if true and false and true and true %}Fail{% else %}Passed{% endif %}`)

	// Shortcut test - the func isn't defined and so will error if it tries to execute
	assertCompileOutput(t,
		"Passed",
		`{% if false and this_shouldnt_be_called_due_to_shortcut() %}Failed{% else %}Passed{% endif %}`)
}

func TestOrCondition(t *testing.T) {
	assertCompileOutput(t,
		"Passed",
		`{% if true or true %}Passed{% else %}Fail{% endif %}`)

	assertCompileOutput(t,
		"Passed",
		`{% if true or true or true or true %}Passed{% else %}Fail{% endif %}`)

	assertCompileOutput(t,
		"Passed",
		`{% if false or true %}Passed{% else %}Failed{% endif %}`)

	assertCompileOutput(t,
		"Passed",
		`{% if true or false %}Passed{% else %}Failed{% endif %}`)

	assertCompileOutput(t,
		"Passed",
		`{% if false or false %}Fail{% else %}Passed{% endif %}`)

	assertCompileOutput(t,
		"Passed",
		`{% if false or false or true or false %}Passed{% else %}Failed{% endif %}`)

	// Shortcut test - the func isn't defined and so will error if it tries to execute
	assertCompileOutput(t,
		"Passed",
		`{% if true or this_shouldnt_be_called_due_to_shortcut() %}Passed{% else %}Failed{% endif %}`)
}

func TestLogicalOperators(t *testing.T) {
	assertCompileOutput(t,
		"Passed",
		`{% if 1 == 1 %}Passed{% else %}Fail{% endif %}`)

	assertCompileOutput(t,
		"Passed",
		`{% if 1 == 2 %}Fail{% else %}Passed{% endif %}`)

	assertCompileOutput(t,
		"Passed",
		`{% if 1 != 2 %}Passed{% else %}Failed{% endif %}`)

	assertCompileOutput(t,
		"Passed",
		`{% if 1 != 1 %}Fail{% else %}Passed{% endif %}`)

	assertCompileOutput(t,
		"Passed",
		`{% if 1 < 2 %}Passed{% else %}Failed{% endif %}`)

	assertCompileOutput(t,
		"Passed",
		`{% if 1 < 1 %}Fail{% else %}Passed{% endif %}`)

	assertCompileOutput(t,
		"Passed",
		`{% if 1 <= 1 %}Passed{% else %}Failed{% endif %}`)

	assertCompileOutput(t,
		"Passed",
		`{% if 1 <= 2 %}Passed{% else %}Failed{% endif %}`)

	assertCompileOutput(t,
		"Passed",
		`{% if 1 <= 0 %}Fail{% else %}Passed{% endif %}`)

	assertCompileOutput(t,
		"Passed",
		`{% if 2 > 1 %}Passed{% else %}Failed{% endif %}`)

	assertCompileOutput(t,
		"Passed",
		`{% if 1 > 2 %}Fail{% else %}Passed{% endif %}`)

	assertCompileOutput(t,
		"Passed",
		`{% if 1 >= 1 %}Passed{% else %}Failed{% endif %}`)

	assertCompileOutput(t,
		"Passed",
		`{% if 2 >= 1 %}Passed{% else %}Failed{% endif %}`)

	assertCompileOutput(t,
		"Passed",
		`{% if 1 > 2 %}Fail{% else %}Passed{% endif %}`)
}

func TestMathOperators(t *testing.T) {
	assertCompileOutput(t, "4", `{{ 1 + 3 }}`)
	assertCompileOutput(t, "-2", `{{ 1 - 3 }}`)
	assertCompileOutput(t, "6", `{{ 2 * 3 }}`)
	assertCompileOutput(t, "9", `{{ 3 ** 2 }}`)
	assertCompileOutput(t, "5", `{{ 10 / 2 }}`)
	assertCompileOutput(t, "2.5", `{{ 5 / 2 }}`)
}

func TestMathOperatorPrecedence(t *testing.T) {
	assertCompileOutput(t, "14", `{{ 2 + 3 * 4 }}`) // Should be parsed as 2 + (3 * 4)
	assertCompileOutput(t, "10", `{{ 2 * 3 + 4 }}`) // Should be parsed as (2 * 3) + 4

	assertCompileOutput(t, "19", `{{   2 + 3  *  4 + 5  }}`) // Should be parsed as 2 + (3 * 4) + 5
	assertCompileOutput(t, "25", `{{  (2 + 3) *  4 + 5  }}`)
	assertCompileOutput(t, "29", `{{   2 + 3  * (4 + 5) }}`)
	assertCompileOutput(t, "45", `{{  (2 + 3) * (4 + 5) }}`)

	assertCompileOutput(t, "-18", `{{  -10 / (20 / 2 ** 2 * 5 / 5) * 8 - 2 }}`)
	assertCompileOutput(t, "41", `{{  10 * 4 - 2 * (4 ** 2 / 4) / 2 / 0.5 + 9 }}`)
}

func TestMathUniaryOperators(t *testing.T) {
	assertCompileOutput(t, "-3", `{{ -3 }}`)
	assertCompileOutput(t, "1", `{{ 4 + -3 }}`)
	assertCompileOutput(t, "-7", `{{ -4 + -3 }}`)
	assertCompileOutput(t, "-1", `{{ -4 - -3 }}`)
}
