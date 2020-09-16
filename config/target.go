package config

import (
	"errors"
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Target struct {
	Name      string
	ProjectID string
	DataSet   string
	Location  string
	Threads   int
}

type Config struct {
	Name   string
	Target *Target
}

var GlobalCfg *Config

func Read(targetProfile string, threads int) (*Config, error) {
	project, err := readDBTProject()
	if err != nil {
		return nil, err
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
		return nil, errors.New(fmt.Sprintf("Output `%s` of profile `%s` not found", targetProfile, project.Profile))
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
		},
	}

	return GlobalCfg, nil
}

func NumberThreads() int {
	return GlobalCfg.Target.Threads
}

type dbtProject struct {
	Name    string `yaml:"name"`
	Profile string `yaml:"profile"`
}

func readDBTProject() (dbtProject, error) {
	project := dbtProject{}

	bytes, err := ioutil.ReadFile("dbt_project.yml")
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
		return dbtProfile{}, errors.New(fmt.Sprintf("dbtProfile `%s` was not found in `profiles.yml`", profileName))
	}

	return p, nil
}
