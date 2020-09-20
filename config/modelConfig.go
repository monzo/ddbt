package config

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
)

type ModelConfig struct {
	Enabled      bool
	Tags         []string
	PreHooks     []string
	PostHooks    []string
	Materialized string
	PersistDocs  struct {
		Relation bool
		Columns  bool
	}
}

var defaultConfig = ModelConfig{
	Enabled:      true,
	Tags:         []string{},
	PreHooks:     []string{},
	PostHooks:    []string{},
	Materialized: "table",
}

var folderBasedConfig = make(map[string]ModelConfig)

func GetFolderConfig(path string) ModelConfig {
	matchPath := ""
	config := defaultConfig

	for cfgPath := range folderBasedConfig {
		if strings.HasPrefix(path, cfgPath) && len(cfgPath) > len(matchPath) {
			matchPath = cfgPath
			config = folderBasedConfig[cfgPath]
		}
	}

	return config
}

func readGeneralFolderBasedConfig(m map[string]interface{}, strExecutor func(s string) (string, error)) error {

	if err := readSubFolder("models/", defaultConfig, m, strExecutor); err != nil {
		return err
	}

	return nil
}

func readSubFolder(folderName string, config ModelConfig, m map[string]interface{}, strExecutor func(s string) (string, error)) error {
	subFolders := make(map[string]map[string]interface{})

	for key, value := range m {
		switch key {
		case "enabled":
			if b, ok := value.(bool); ok {
				config.Enabled = b
			} else {
				return errors.New(fmt.Sprintf("Unable to convert `enabled` to boolean, got: %v", reflect.TypeOf(value)))
			}

		case "tags":
			list, err := strOrList("tags", value, strExecutor)
			if err != nil {
				return err
			}
			config.Tags = list

		case "pre_hook":
			list, err := strOrList("pre_hook", value, strExecutor)
			if err != nil {
				return err
			}
			config.PreHooks = list

		case "post_hook":
			list, err := strOrList("post_hook", value, strExecutor)
			if err != nil {
				return err
			}
			config.PostHooks = list

		case "database":
			_, err := asStr("database", value, strExecutor)
			if err != nil {
				return err
			}

		case "schema":
			_, err := asStr("schema", value, strExecutor)
			if err != nil {
				return err
			}

		case "persist_docs":

		case "materialized":
			materialized, err := asStr("materialized", value, strExecutor)
			if err != nil {
				return err
			}

			config.Materialized = materialized

		default:
			if v, ok := value.(map[interface{}]interface{}); ok {
				strMap := make(map[string]interface{})

				for k, v := range v {
					kStr, ok := k.(string)
					if !ok {
						return errors.New(fmt.Sprintf("unable to convert key `%v` into a string", key))
					}

					strMap[kStr] = v
				}

				subFolders[key] = strMap
			} else {
				return errors.New(fmt.Sprintf("unable to convert `%s` into a map, got; %v", key, reflect.TypeOf(value)))
			}
		}
	}

	for name, value := range subFolders {
		if err := readSubFolder(fmt.Sprintf("%s%s%c", folderName, name, os.PathSeparator), config, value, strExecutor); err != nil {
			return err
		}
	}

	folderBasedConfig[folderName] = config

	return nil
}

func strOrList(name string, value interface{}, strExecutor func(s string) (string, error)) ([]string, error) {
	switch v := value.(type) {
	case string:
		return []string{v}, nil

	case []string:
		return v, nil

	case []interface{}:
		list := make([]string, len(v))

		for i, value := range v {
			str, err := asStr(name, value, strExecutor)
			if err != nil {
				return nil, err
			}

			list[i] = str
		}

		return list, nil

	default:
		return nil, errors.New(fmt.Sprintf("Unable to convert into a list of strings for `%s`, got %v", name, reflect.TypeOf(value)))
	}
}

func asStr(name string, value interface{}, strExecutor func(s string) (string, error)) (string, error) {
	strValue, ok := value.(string)
	if !ok {
		return "", errors.New(fmt.Sprintf("Unable to convert `%s` to string, got: %v", name, reflect.TypeOf(value)))
	}

	strValue, err := strExecutor(strValue)
	if err != nil {
		return "", err
	}

	return strValue, nil
}
