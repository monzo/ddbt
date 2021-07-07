package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"

	"gopkg.in/yaml.v2"
)

// Model Groups are a custom addition to DBT which allow us to specify different configurations depending on the folder
// the model lives in. This allows us to store all our analytics in a single monorepo and run them as part of the same
// DAG

type modelGroupTarget struct {
	Project           string      `yaml:"project"`
	Dataset           interface{} `yaml:"dataset"`            // This could either be a string, or a map
	ExecutionProjects []string    `yaml:"execution_projects"` // Which projects to run the queries under

	// Here, project-tag substitutions can be added. Models with matching tags and projects will have the
	// destination project swapped.
	ProjectSubstitutions map[string]map[string]string `yaml:"project_tag_substitutions"`
}

type modelGroupConfig struct {
	Targets map[string]modelGroupTarget
}

type modelGroupConfigFile = map[string]modelGroupConfig

func readModelGroupConfig(fileName string, targetName string, defaultTarget string, baseTarget *Target) (map[string]*Target, error) {
	m := make(modelGroupConfigFile)

	bytes, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(bytes, &m); err != nil {
		return nil, err
	}

	rtn := make(map[string]*Target)

	for modelGroup, targets := range m {
		targetCfg, found := targets.Targets[targetName]
		if !found {
			fmt.Printf("⚠️ Model group `%s` does not have a target for `%s`\n", modelGroup, targetName)
			continue
		}

		target := baseTarget.Copy()

		if err := updateTargetFromModelGroupConfig(target, targetName, targetCfg); err != nil {
			return nil, err
		}

		if defaultTarget != "" {
			defaultTargetCfg, found := targets.Targets[defaultTarget]
			if !found {
				fmt.Printf("⚠️ Model group `%s` does not have a target for `%s`\n", modelGroup, targetName)
				continue
			}

			if err := updateTargetFromModelGroupConfig(target.ReadUpstream, defaultTarget, defaultTargetCfg); err != nil {
				return nil, err
			}
		}

		rtn[modelGroup] = target
	}

	return rtn, nil
}

func updateTargetFromModelGroupConfig(target *Target, targetName string, targetCfg modelGroupTarget) error {
	if targetCfg.Project != "" {
		target.ProjectID = targetCfg.Project
	}

	// Data set is either a string, or "from_env" which means it's auto generated using the users username
	if targetCfg.Dataset != nil {
		switch v := targetCfg.Dataset.(type) {
		case string:
			target.DataSet = v

		case map[interface{}]interface{}:
			if b, ok := v["from_env"].(bool); ok && b {
				u, err := user.Current()
				if err != nil {
					return err
				}
				overrideUserDataset := os.Getenv("OVERRIDE_USER_DATASET")
				var username string
				if overrideUserDataset != "" {
					username = overrideUserDataset
				} else {
					username = u.Username
				}
				target.DataSet = fmt.Sprintf("dbt_%s_%s", username, targetName)
			} else {
				return errors.New("expected dataset to be string or { 'from_env': true }")
			}

		default:
			return errors.New("expected dataset to be string or { 'from_env': true }")
		}
	}

	if targetCfg.ExecutionProjects != nil {
		target.ExecutionProjects = targetCfg.ExecutionProjects
	}

	if targetCfg.ProjectSubstitutions != nil {
		target.ProjectSubstitutions = targetCfg.ProjectSubstitutions
	}

	return nil
}
