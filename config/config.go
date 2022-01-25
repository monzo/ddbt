package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	// "ddbt/cmd"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Name   string
	Target *Target

	// Custom behaviour which allows us to override the target information on a per folder basis within `/models/`
	ModelGroups     map[string]*Target
	ModelGroupsFile string

	// seedConfig holds the seed (global) configurations
	seedConfig map[string]*SeedConfig
}

func (c *Config) GetTargetFor(path string) *Target {
	if c.ModelGroups == nil {
		return c.Target
	}

	for modelGroup, target := range c.ModelGroups {
		if strings.HasPrefix(path, fmt.Sprintf("models%c%s%c", os.PathSeparator, modelGroup, os.PathSeparator)) {
			return target
		}

		if strings.HasPrefix(path, fmt.Sprintf("tests%c%s%c", os.PathSeparator, modelGroup, os.PathSeparator)) {
			return target
		}

		if strings.HasPrefix(path, fmt.Sprintf("data%c%s%c", os.PathSeparator, modelGroup, os.PathSeparator)) {
			return target
		}
	}

	return c.Target
}

var GlobalCfg *Config

func Read(targetProfile string, upstreamProfile string, threads int, customConfigPath string, strExecutor func(s string) (string, error)) (*Config, error) {
	project, err := readDBTProject(customConfigPath)
	if err != nil {
		return nil, err
	}

	appConfig, err := readDDBTConfig()
	if err != nil {
		return nil, err
	}

	for _, target := range appConfig.ProtectedTargets {
		if strings.EqualFold(target, targetProfile) {
			return nil, fmt.Errorf("`%s` is a protected target, DDBT will not run against it.", target)
		}
	}

	profile, err := readProfile(project.Profile)
	if err != nil {
		return nil, err
	}

	if targetProfile == "" {
		targetProfile = profile.Target
	}

	output, found := profile.Outputs[targetProfile]
	if !found {
		return nil, fmt.Errorf("Output `%s` of profile `%s` not found", targetProfile, project.Profile)
	}

	if threads <= 0 {
		threads = output.Threads
	}

	GlobalCfg = &Config{
		Name: project.Name,
		Target: &Target{
			Name:      targetProfile,
			ProjectID: output.Project,
			DataSet:   output.Dataset,
			Location:  output.Location,
			Threads:   threads,

			ProjectSubstitutions: make(map[string]map[string]string),
			ExecutionProjects:    make([]string, 0),
		},
	}

	if upstreamProfile != "" {
		output, found := profile.Outputs[upstreamProfile]
		if !found {
			return nil, fmt.Errorf("Output `%s` of profile `%s` not found", upstreamProfile, project.Profile)
		}

		GlobalCfg.Target.ReadUpstream = &Target{
			Name:      upstreamProfile,
			ProjectID: output.Project,
			DataSet:   output.Dataset,
			Location:  output.Location,
			Threads:   threads,

			ProjectSubstitutions: make(map[string]map[string]string),
			ExecutionProjects:    make([]string, 0),
		}
	}

	if appConfig.ModelGroupsFile != "" {
		modelGroups, err := readModelGroupConfig(appConfig.ModelGroupsFile, targetProfile, upstreamProfile, GlobalCfg.Target)
		if err != nil {
			return nil, err
		}

		GlobalCfg.ModelGroups = modelGroups
		GlobalCfg.ModelGroupsFile = appConfig.ModelGroupsFile
	}

	if settings, found := project.Models[project.Name]; found {
		if err := readGeneralFolderBasedConfig(settings, strExecutor); err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("no models config found, expected to find `models: %s:` in `dbt_project.yml`", project.Name)
	}

	if seedCfg, found := project.Seeds[project.Name]; found {
		cfg, err := readSeedCfg(seedCfg)
		if err != nil {
			// if parsing of seed section of config has failed, don't error
			fmt.Fprintf(os.Stderr, "⚠️ Cannot parse seed config: %v\n", err)
		} else {
			GlobalCfg.seedConfig = cfg
		}
	}

	return GlobalCfg, nil
}

func NumberThreads() int {
	return GlobalCfg.Target.Threads
}

type dbtProject struct {
	Name    string                            `yaml:"name"`
	Profile string                            `yaml:"profile"`
	Models  map[string]map[string]interface{} `yaml:"models"` // "Models[project_name][key]value"
	Seeds   map[string]map[string]interface{} `yaml:"seeds"`  // "Seeds[project_name][key]value"
}

func readDBTProject(customConfigPath string) (dbtProject, error) {
	project := dbtProject{}

	bytes, err := ioutil.ReadFile(customConfigPath + "dbt_project.yml")
	if err != nil {
		return dbtProject{}, err
	}

	if err := yaml.Unmarshal(bytes, &project); err != nil {
		return dbtProject{}, err
	}

	return project, nil
}

type dbtOutputs struct {
	Project  string `yaml:"project"`
	Dataset  string `yaml:"dataset"`
	Location string `yaml:"location"`
	Threads  int    `yaml:"threads"`
}

type dbtProfile struct {
	Target  string `yaml:"target"`
	Outputs map[string]dbtOutputs
}

func readProfile(profileName string) (dbtProfile, error) {
	m := make(map[string]dbtProfile)

	bytes, err := ioutil.ReadFile("profiles.yml")
	if err != nil {
		return dbtProfile{}, err
	}

	if err := yaml.Unmarshal(bytes, &m); err != nil {
		return dbtProfile{}, err
	}

	p, found := m[profileName]
	if !found {
		return dbtProfile{}, fmt.Errorf("dbtProfile `%s` was not found in `profiles.yml`", profileName)
	}

	return p, nil
}

type ddbtConfig struct {
	ModelGroupsFile  string   `yaml:"model-groups-config"`
	ProtectedTargets []string `yaml:"protected-targets"` // Targets that DDBT is not allowed to execute against
}

func readDDBTConfig() (ddbtConfig, error) {
	c := ddbtConfig{}

	bytes, err := ioutil.ReadFile("ddbt_config.yml")
	if os.IsNotExist(err) {
		return c, nil
	}
	if err != nil {
		return c, err
	}

	if err := yaml.Unmarshal(bytes, &c); err != nil {
		return c, err
	}

	return c, nil
}
