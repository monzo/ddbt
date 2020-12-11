package config

import (
	"testing"

	"gopkg.in/yaml.v2"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeedConfig(t *testing.T) {
	dbtProjectYml := `
name: package_name
version: '1.0'

profile: ddbt
seeds:
  ddbt:
    enabled: true
    schema: default_dataset_name
    table_name:
      column_types:
        id: string
        amount: numeric
        description: string
      subfolder_table_name:
        column_types:
          extra_field: string
`

	var project dbtProject
	require.NoError(t, yaml.Unmarshal([]byte(dbtProjectYml), &project))
	seedCfg, err := readSeedCfg(project.Seeds["ddbt"])
	require.NoError(t, err)

	assert.NotNil(t, seedCfg["data/table_name"])
	assert.Equal(t, map[string]string{"id": "string", "amount": "numeric", "description": "string"}, seedCfg["data/table_name"].ColumnTypes)

	assert.NotNil(t, seedCfg["data/table_name/subfolder_table_name"])
	assert.Equal(t, map[string]string{"extra_field": "string"}, seedCfg["data/table_name/subfolder_table_name"].ColumnTypes)
}

func TestSeedConfigParsing(t *testing.T) {
	// Valid examples from docs.getdbt.com
	tcs := [...]struct {
		name          string
		dbtProjectYml string
	}{
		{
			name: "Apply the schema configuration to all seeds in your project",
			dbtProjectYml: `
seeds:
  jaffle_shop:
    schema: seed_data
`,
		},
		{
			name: "Apply the schema configuration to one seed only",
			dbtProjectYml: `
seeds:
  jaffle_shop:
    marketing:
      utm_parameters:
        schema: seed_data
`,
		},
		{
			name: "Example seed configuration",
			dbtProjectYml: `
name: jaffle_shop

seeds:
  jaffle_shop:
    enabled: true
    schema: seed_data
    # This configures data/country_codes.csv
    country_codes:
      # Override column types
      column_types:
        country_code: varchar(2)
        country_name: varchar(32)
    marketing:
      schema: marketing # this will take precedence
`,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var project dbtProject
			require.NoError(t, yaml.Unmarshal([]byte(tc.dbtProjectYml), &project))
			_, err := readSeedCfg(project.Seeds["jaffle_shop"])
			require.NoError(t, err)
		})
	}

}
