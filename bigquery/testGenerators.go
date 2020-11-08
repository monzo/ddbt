package bigquery

import (
	"fmt"

	"ddbt/properties"
)

// Given a schema test, will generate BigQuery standard SQL to perform that test
func GenerateTestSQL(test *properties.Test) (sql string, err error) {
	return fmt.Sprintf("{{ test_%s() }}", test.Name), nil
	//generator, found := testGenerators[test.Name]
	//if !found {
	//	return "", errors.New(fmt.Sprintf("Unknown test type `%s`", test.Name))
	//}
	//
	//return generator(test)
}

var testGenerators = map[string]func(test *properties.Test) (sql string, err error){}
