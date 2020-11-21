package fs

import (
	"ddbt/config"
	"path/filepath"
	"strings"
)

// SeedFile is a simplified File where we only keep track of its name and path.
type SeedFile struct {
	Name string
	Path string
}

func newSeedFile(path string) *SeedFile {
	return &SeedFile{
		Name: strings.TrimSuffix(filepath.Base(path), ".csv"),
		Path: path,
	}
}

func (s SeedFile) GetName() string {
	return s.Name
}

func (s SeedFile) GetTarget() (*config.Target, error) {
	// No target overrides
	return config.GlobalCfg.GetTargetFor(s.Path), nil
}

func (s SeedFile) GetConfig() (*config.SeedConfig, error) {
	return config.GlobalCfg.GetSeedConfig(s.Path), nil
}
