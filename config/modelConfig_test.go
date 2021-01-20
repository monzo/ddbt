package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestModelConfig(t *testing.T) {
	dbtProjectYml := `
name: package_name
version: '1.0'

profile: ddbt
models:
  ddbt:
    enabled: true
    schema: default_dataset_name
    tags: ["tag_one", "tag_two"]
    materialized: ephemeral
    table_name:
      tags: # General Config
        - tag_two
        - tag_three
      materialized: table # Model config
    another_table_name:
      tags: []
`

	var project dbtProject
	require.NoError(t, yaml.Unmarshal([]byte(dbtProjectYml), &project))
	err := readGeneralFolderBasedConfig(project.Models["ddbt"], func(s string) (string, error) { return s, nil })
	require.NoError(t, err)

	assert.NotNil(t, folderBasedConfig["models/"])
	assert.Equal(t, []string{"tag_one", "tag_two"}, folderBasedConfig["models/"].Tags)
	assert.Equal(t, "ephemeral", folderBasedConfig["models/"].Materialized)

	// Override parent materialized
	assert.NotNil(t, folderBasedConfig["models/table_name/"])
	assert.Equal(t, []string{"tag_two", "tag_three"}, folderBasedConfig["models/table_name/"].Tags)
	assert.Equal(t, "table", folderBasedConfig["models/table_name/"].Materialized)

	// Inherit materialized from parent
	assert.NotNil(t, folderBasedConfig["models/another_table_name/"])
	assert.Equal(t, []string(nil), folderBasedConfig["models/another_table_name"].Tags)
	assert.Equal(t, "ephemeral", folderBasedConfig["models/another_table_name/"].Materialized)
}
