package config

type Target struct {
	Name      string
	ProjectID string
	DataSet   string
}

type Config struct {
	Name   string
	Target *Target
}

var GlobalCfg *Config

func Read() *Config {
	GlobalCfg = &Config{
		Name: "Test Project",
		Target: &Target{
			Name:      "dev",
			ProjectID: "SOME-PROJECT", //FIXME
			DataSet:   "SOME-DATASET", //FIXME
		},
	}

	return GlobalCfg
}
