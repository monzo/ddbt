package tests

import (
	"testing"
)

func TestUDFEscapedQuotes(t *testing.T) {
	const udf = "test1 \\'test2\\' \\\\\\ "
	assertCompileOutput(t, "test1 'test2' \\\\ test3", "{{ config(udf='''"+udf+"''') }}test3")
}
