package config

import (
	"fmt"
	"os"
	"strings"
)

// SeedConfig represents the seed configuration specified in dbt_project.yml
type SeedConfig struct {
	GeneralConfig
	QuoteColumns bool              `yaml:"quote_columns"`
	ColumnTypes  map[string]string `yaml:"column_types"`
}

// GetSeedConfig returns the seed configuration for a given path.
func (c *Config) GetSeedConfig(path string) *SeedConfig {
	return c.GetFolderBasedSeedConfig(path)
}

// GetFolderBasedSeedConfig is a version of GetFolderConfig to
// apply parent seed config hierarchically to child folders.
func (c *Config) GetFolderBasedSeedConfig(path string) *SeedConfig {
	configPath := "data"
	parentConfig := c.seedConfig[configPath]
	config := &SeedConfig{
		GeneralConfig: GeneralConfig{
			Enabled: parentConfig.Enabled,
			Schema:  parentConfig.Schema,
		},
		QuoteColumns: parentConfig.QuoteColumns,
		ColumnTypes:  parentConfig.ColumnTypes,
	}

	parts := strings.Split(strings.TrimSuffix(path, ".csv"), string(os.PathSeparator))
	for _, part := range parts[1:] {
		configPath = fmt.Sprintf("%s%c%s", configPath, os.PathSeparator, part)
		childConfig := c.seedConfig[configPath]
		if childConfig != nil {
			if childConfig.ColumnTypes != nil {
				config.ColumnTypes = childConfig.ColumnTypes
			}
			if childConfig.Schema != "" {
				config.Schema = childConfig.Schema
			}
			if childConfig.Enabled != config.Enabled {
				config.Enabled = childConfig.Enabled
			}
		}
	}

	return config
}

func readSeedCfg(seedCfg map[string]interface{}) (map[string]*SeedConfig, error) {
	parser := &seedConfigParser{
		SeedConfigs: make(map[string]*SeedConfig),
	}
	if err := parser.readCfgForDir("data", seedCfg); err != nil {
		return nil, err
	}
	return parser.SeedConfigs, nil
}

type seedConfigParser struct {
	SeedConfigs map[string]*SeedConfig
}

func (p *seedConfigParser) readCfgForDir(pathPrefix string, seedCfg map[string]interface{}) error {
	subFolders := make(map[string]map[string]interface{})
	var cfg SeedConfig

	// Process common general configurations.
	remaining, err := readGeneralConfig(&cfg.GeneralConfig, seedCfg, simpleStringExecutor)
	if err != nil {
		return err
	}

	for key, value := range remaining {
		switch key {
		case "quote_columns":
			b, err := asBool(key, value)
			if err != nil {
				return err
			}
			cfg.QuoteColumns = b

		case "column_types":
			kvm, err := asKeyValueStringMap(key, value)
			if err != nil {
				return err
			}
			cfg.ColumnTypes = kvm

		default:
			genericMap, ok := value.(map[interface{}]interface{})
			if !ok {
				return fmt.Errorf("Unable to convert `%s` to map, got: %T", key, value)
			}

			kvm := make(map[string]interface{})
			for k, v := range genericMap {
				kStr, ok := k.(string)
				if !ok {
					return fmt.Errorf("Unable to convert key `%v` to string", k)
				}
				kvm[kStr] = v
			}

			subFolders[key] = kvm
		}
	}

	// Recurse into sub folders
	for name, value := range subFolders {
		if err := p.readCfgForDir(fmt.Sprintf("%s%c%s", pathPrefix, os.PathSeparator, name), value); err != nil {
			return err
		}
	}

	p.SeedConfigs[pathPrefix] = &cfg

	return nil
}

func simpleStringExecutor(s string) (string, error) {
	return s, nil
}

// asKeyValueStringMap converts value to a map[string]string.
func asKeyValueStringMap(key, value interface{}) (map[string]string, error) {
	genericMap, ok := value.(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf("Unable to convert `%s` to map, got %T", key, value)
	}

	kvm := make(map[string]string)
	for k, v := range genericMap {
		kStr, ok := k.(string)
		if !ok {
			return nil, fmt.Errorf("Unable to convert key `%s` to string, got: %T", k, k)
		}
		vStr, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("Unable to convert value `%s` to string, got: %T", v, v)
		}
		kvm[kStr] = vStr
	}
	return kvm, nil
}

// asBool converts value to a bool.
func asBool(key string, value interface{}) (bool, error) {
	b, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("Unable to convert `%s` to boolean, got: %T", key, value)
	}
	return b, nil
}
