package tests

import "testing"

func TestIsBoolean(t *testing.T) {
	assertCompileOutput(t,
		`PassPassPassPass`,
		`
{%- set a = false -%}
{%- set b = 0 -%}
{%- if a is boolean -%}Pass{% else %}Fail{% endif -%}
{%- if a is not boolean -%}Fail{% else %}Pass{% endif -%}
{%- if b is boolean -%}Fail{% else %}Pass{% endif -%}
{%- if b is not boolean -%}Pass{% else %}Fail{% endif -%}
`)
}

func TestIsCallable(t *testing.T) {
	assertCompileOutput(t,
		`PassPassPassPass`,
		`
{%- macro a() -%}{%- endmacro -%}
{%- set b = 0 -%}
{%- if a is callable -%}Pass{% else %}Fail{% endif -%}
{%- if a is not callable -%}Fail{% else %}Pass{% endif -%}
{%- if b is callable -%}Fail{% else %}Pass{% endif -%}
{%- if b is not callable -%}Pass{% else %}Fail{% endif -%}
`)
}

func TestIsDivisibleBy(t *testing.T) {
	assertCompileOutput(t,
		`PassPassPassPass`,
		`
{%- set a = 10 -%}
{%- if a is divisibleby(2) -%}Pass{% else %}Fail{% endif -%}
{%- if a is not divisibleby(2) -%}Fail{% else %}Pass{% endif -%}
{%- if a is divisibleby(3) -%}Fail{% else %}Pass{% endif -%}
{%- if a is not divisibleby(3) -%}Pass{% else %}Fail{% endif -%}
`)
}

func TestIsEq(t *testing.T) {
	aliases := []string{"eq", "==", "equalto"}

	for _, alias := range aliases {
		alias := alias

		t.Run(alias, func(t *testing.T) {
			assertCompileOutput(t,
				`PassPassPassPass`,
				`
{%- set a = 34 -%}
{%- set b = 45 -%}
{%- if a is `+alias+`(34) -%}Pass{% else %}Fail{% endif -%}
{%- if a is not `+alias+`(34) -%}Fail{% else %}Pass{% endif -%}
{%- if b is `+alias+`(a) -%}Fail{% else %}Pass{% endif -%}
{%- if b is not `+alias+`(a) -%}Pass{% else %}Fail{% endif -%}
`)
		})
	}
}

func TestIsEven(t *testing.T) {
	assertCompileOutput(t,
		`PassPassPassPass`,
		`
{%- set a = 34 -%}
{%- set b = 45 -%}
{%- if a is even -%}Pass{% else %}Fail{% endif -%}
{%- if a is not even -%}Fail{% else %}Pass{% endif -%}
{%- if b is even -%}Fail{% else %}Pass{% endif -%}
{%- if b is not even -%}Pass{% else %}Fail{% endif -%}
`)
}

func TestIsFalse(t *testing.T) {
	assertCompileOutput(t,
		`PassPassPassPass`,
		`
{%- set a = false -%}
{%- set b = true -%}
{%- if a is false -%}Pass{% else %}Fail{% endif -%}
{%- if a is not false -%}Fail{% else %}Pass{% endif -%}
{%- if b is false -%}Fail{% else %}Pass{% endif -%}
{%- if b is not false -%}Pass{% else %}Fail{% endif -%}
`)
}

func TestIsFloat(t *testing.T) {
	assertCompileOutput(t,
		`PassPassPassPass`,
		`
{%- set a = 43.34 -%}
{%- set b = 43 -%}
{%- if a is float -%}Pass{% else %}Fail{% endif -%}
{%- if a is not float -%}Fail{% else %}Pass{% endif -%}
{%- if b is float -%}Fail{% else %}Pass{% endif -%}
{%- if b is not float -%}Pass{% else %}Fail{% endif -%}
`)
}

func TestIsGE(t *testing.T) {
	aliases := []string{"ge", ">="}

	for _, alias := range aliases {
		alias := alias

		t.Run(alias, func(t *testing.T) {
			assertCompileOutput(t,
				`PassPassPassPass`,
				`
{%- set a = 30 -%}
{%- set b = 29 -%}
{%- if a is `+alias+`(30) -%}Pass{% else %}Fail{% endif -%}
{%- if a is not `+alias+`(30) -%}Fail{% else %}Pass{% endif -%}
{%- if b is `+alias+`(30) -%}Fail{% else %}Pass{% endif -%}
{%- if b is not `+alias+`(30) -%}Pass{% else %}Fail{% endif -%}
`)
		})
	}
}

func TestIsGT(t *testing.T) {
	aliases := []string{"gt", ">", "greaterthan"}

	for _, alias := range aliases {
		alias := alias

		t.Run(alias, func(t *testing.T) {
			assertCompileOutput(t,
				`PassPassPassPass`,
				`
{%- set a = 31 -%}
{%- set b = 30 -%}
{%- if a is `+alias+`(30) -%}Pass{% else %}Fail{% endif -%}
{%- if a is not `+alias+`(30) -%}Fail{% else %}Pass{% endif -%}
{%- if b is `+alias+`(30) -%}Fail{% else %}Pass{% endif -%}
{%- if b is not `+alias+`(30) -%}Pass{% else %}Fail{% endif -%}
`)
		})
	}
}

func TestIsIn(t *testing.T) {
	assertCompileOutput(t,
		`PassPassPassPass`,
		`
{%- set a = 4 -%}
{%- set b = 23 -%}
{%- set c = [1,2,3,4,5] -%}
{%- if a is in(c) -%}Pass{% else %}Fail{% endif -%}
{%- if a is not in(c) -%}Fail{% else %}Pass{% endif -%}
{%- if b is in(c) -%}Fail{% else %}Pass{% endif -%}
{%- if b is not in(c) -%}Pass{% else %}Fail{% endif -%}
`)
}

func TestIsInteger(t *testing.T) {
	assertCompileOutput(t,
		`PassPassPassPass`,
		`
{%- set a = 42 -%}
{%- set b = 42.123 -%}
{%- if a is integer -%}Pass{% else %}Fail{% endif -%}
{%- if a is not integer -%}Fail{% else %}Pass{% endif -%}
{%- if b is integer -%}Fail{% else %}Pass{% endif -%}
{%- if b is not integer -%}Pass{% else %}Fail{% endif -%}
`)
}

func TestIsIterable(t *testing.T) {
	assertCompileOutput(t,
		`PassPassPassPassPassPass`,
		`
{%- set a = [] -%}
{%- set b = {} -%}
{%- set c = false -%}
{%- if a is iterable -%}Pass{% else %}Fail{% endif -%}
{%- if a is not iterable -%}Fail{% else %}Pass{% endif -%}
{%- if b is iterable -%}Pass{% else %}Fail{% endif -%}
{%- if b is not iterable -%}Fail{% else %}Pass{% endif -%}
{%- if c is iterable -%}Fail{% else %}Pass{% endif -%}
{%- if c is not iterable -%}Pass{% else %}Fail{% endif -%}
`)
}

func TestIsLE(t *testing.T) {
	aliases := []string{"le", "<="}

	for _, alias := range aliases {
		alias := alias

		t.Run(alias, func(t *testing.T) {
			assertCompileOutput(t,
				`PassPassPassPass`,
				`
{%- set a = 30 -%}
{%- set b = 31 -%}
{%- if a is `+alias+`(30) -%}Pass{% else %}Fail{% endif -%}
{%- if a is not `+alias+`(30) -%}Fail{% else %}Pass{% endif -%}
{%- if b is `+alias+`(30) -%}Fail{% else %}Pass{% endif -%}
{%- if b is not `+alias+`(30) -%}Pass{% else %}Fail{% endif -%}
`)
		})
	}
}

func TestIsLower(t *testing.T) {
	assertCompileOutput(t,
		`PassPassPassPass`,
		`
{%- set a = 'hello world' -%}
{%- set b = 'hello World' -%}
{%- if a is lower -%}Pass{% else %}Fail{% endif -%}
{%- if a is not lower -%}Fail{% else %}Pass{% endif -%}
{%- if b is lower -%}Fail{% else %}Pass{% endif -%}
{%- if b is not lower -%}Pass{% else %}Fail{% endif -%}
`)
}

func TestIsLT(t *testing.T) {
	aliases := []string{"lt", "<", "lessthan"}

	for _, alias := range aliases {
		alias := alias

		t.Run(alias, func(t *testing.T) {
			assertCompileOutput(t,
				`PassPassPassPass`,
				`
{%- set a = 29 -%}
{%- set b = 30 -%}
{%- if a is `+alias+`(30) -%}Pass{% else %}Fail{% endif -%}
{%- if a is not `+alias+`(30) -%}Fail{% else %}Pass{% endif -%}
{%- if b is `+alias+`(30) -%}Fail{% else %}Pass{% endif -%}
{%- if b is not `+alias+`(30) -%}Pass{% else %}Fail{% endif -%}
`)
		})
	}
}

func TestIsMapping(t *testing.T) {
	assertCompileOutput(t,
		`PassPassPassPass`,
		`
{%- set a = {} -%}
{%- set b = [] -%}
{%- if a is mapping -%}Pass{% else %}Fail{% endif -%}
{%- if a is not mapping -%}Fail{% else %}Pass{% endif -%}
{%- if b is mapping -%}Fail{% else %}Pass{% endif -%}
{%- if b is not mapping -%}Pass{% else %}Fail{% endif -%}
`)
}

func TestIsNE(t *testing.T) {
	aliases := []string{"ne", "!="}

	for _, alias := range aliases {
		alias := alias

		t.Run(alias, func(t *testing.T) {
			assertCompileOutput(t,
				`PassPassPassPass`,
				`
{%- set a = 29 -%}
{%- set b = 30 -%}
{%- if a is `+alias+`(30) -%}Pass{% else %}Fail{% endif -%}
{%- if a is not `+alias+`(30) -%}Fail{% else %}Pass{% endif -%}
{%- if b is `+alias+`(30) -%}Fail{% else %}Pass{% endif -%}
{%- if b is not `+alias+`(30) -%}Pass{% else %}Fail{% endif -%}
`)
		})
	}
}

func TestIsNone(t *testing.T) {
	assertCompileOutput(t,
		`PassPassPassPass`,
		`
{%- set a = None -%}
{%- set b = true -%}
{%- if a is none -%}Pass{% else %}Fail{% endif -%}
{%- if a is not none -%}Fail{% else %}Pass{% endif -%}
{%- if b is none -%}Fail{% else %}Pass{% endif -%}
{%- if b is not none -%}Pass{% else %}Fail{% endif -%}
`)
}

func TestIsNumber(t *testing.T) {
	assertCompileOutput(t,
		`PassPassPassPassPassPass`,
		`
{%- set a = 43.34 -%}
{%- set b = 43 -%}
{%- set c = true -%}
{%- if a is number -%}Pass{% else %}Fail{% endif -%}
{%- if a is not number -%}Fail{% else %}Pass{% endif -%}
{%- if b is number -%}Pass{% else %}Fail{% endif -%}
{%- if b is not number -%}Fail{% else %}Pass{% endif -%}
{%- if c is number -%}Fail{% else %}Pass{% endif -%}
{%- if c is not number -%}Pass{% else %}Fail{% endif -%}
`)
}

func TestIsOdd(t *testing.T) {
	assertCompileOutput(t,
		`PassPassPassPass`,
		`
{%- set a = 33 -%}
{%- set b = 46 -%}
{%- if a is odd -%}Pass{% else %}Fail{% endif -%}
{%- if a is not odd -%}Fail{% else %}Pass{% endif -%}
{%- if b is odd -%}Fail{% else %}Pass{% endif -%}
{%- if b is not odd -%}Pass{% else %}Fail{% endif -%}
`)
}

func TestIsSameAs(t *testing.T) {
	assertCompileOutput(t,
		`PassPassPassPass`,
		`
{%- set a = 33 -%}
{%- set b = 33 -%}
{%- if a is sameas(a) -%}Pass{% else %}Fail{% endif -%}
{%- if a is not sameas(a) -%}Fail{% else %}Pass{% endif -%}
{%- if a is sameas(b) -%}Fail{% else %}Pass{% endif -%}
{%- if a is not sameas(b) -%}Pass{% else %}Fail{% endif -%}
`)
}

func TestIsSequence(t *testing.T) {
	assertCompileOutput(t,
		`PassPassPassPassPassPass`,
		`
{%- set a = [] -%}
{%- set b = {} -%}
{%- set c = 'hello world' -%}
{%- if a is sequence -%}Pass{% else %}Fail{% endif -%}
{%- if a is not sequence -%}Fail{% else %}Pass{% endif -%}
{%- if b is sequence -%}Pass{% else %}Fail{% endif -%}
{%- if b is not sequence -%}Fail{% else %}Pass{% endif -%}
{%- if c is sequence -%}Fail{% else %}Pass{% endif -%}
{%- if c is not sequence -%}Pass{% else %}Fail{% endif -%}
`)
}

func TestIsString(t *testing.T) {
	assertCompileOutput(t,
		`PassPassPassPass`,
		`
{%- set a = 'hello world' -%}
{%- set b = 123 -%}
{%- if a is string -%}Pass{% else %}Fail{% endif -%}
{%- if a is not string -%}Fail{% else %}Pass{% endif -%}
{%- if b is string -%}Fail{% else %}Pass{% endif -%}
{%- if b is not string -%}Pass{% else %}Fail{% endif -%}
`)
}

func TestIsTrue(t *testing.T) {
	assertCompileOutput(t,
		`PassPassPassPass`,
		`
{%- set a = true -%}
{%- set b = false -%}
{%- if a is true -%}Pass{% else %}Fail{% endif -%}
{%- if a is not true -%}Fail{% else %}Pass{% endif -%}
{%- if b is true -%}Fail{% else %}Pass{% endif -%}
{%- if b is not true -%}Pass{% else %}Fail{% endif -%}
`)
}

func TestIsUndefined(t *testing.T) {
	assertCompileOutput(t,
		`PassPassPassPass`,
		`
{%- set a = None -%}
{%- set b = Null -%}
{%- if a is undefined -%}Pass{% else %}Fail{% endif -%}
{%- if a is not undefined -%}Fail{% else %}Pass{% endif -%}
{%- if b is undefined -%}Fail{% else %}Pass{% endif -%}
{%- if b is not undefined -%}Pass{% else %}Fail{% endif -%}
`)
}

func TestIsUpper(t *testing.T) {
	assertCompileOutput(t,
		`PassPassPassPass`,
		`
{%- set a = 'HELLO WORLD' -%}
{%- set b = 'HELLO WorlD' -%}
{%- if a is upper -%}Pass{% else %}Fail{% endif -%}
{%- if a is not upper -%}Fail{% else %}Pass{% endif -%}
{%- if b is upper -%}Fail{% else %}Pass{% endif -%}
{%- if b is not upper -%}Pass{% else %}Fail{% endif -%}
`)
}
