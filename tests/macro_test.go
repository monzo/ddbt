package tests

import "testing"

func TestBasicMacro(t *testing.T) {
	assertCompileOutput(t, "10\nhello world!\n",
		`
{%- macro concat(a, b) %}{{ a ~ b }}{% endmacro -%}
{{ concat(1, 0) }}
{{ concat(concat("hello", " "), concat("world", "!")) }}
`)
}

func TestCallerMacro(t *testing.T) {
	assertCompileOutput(t, "30\n35\n",
		`
{%- macro add(a) %}{{ a + caller() }}{% endmacro -%}
{% call add(5) %}25{% endcall %}
{% call add(5) %}{{25 + a }}{% endcall %}
`)
}
