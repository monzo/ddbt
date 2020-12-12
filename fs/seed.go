package fs

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"ddbt/config"
)

// SeedFile is a simplified File where we only keep track of its name and path.
type SeedFile struct {
	Name        string
	Path        string
	Columns     []string
	ColumnTypes map[string]string
}

func newSeedFile(path string) *SeedFile {
	return &SeedFile{
		Name: strings.TrimSuffix(filepath.Base(path), ".csv"),
		Path: path,
	}
}

func (s *SeedFile) GetName() string {
	return s.Name
}

func (s *SeedFile) GetTarget() (*config.Target, error) {
	target := config.GlobalCfg.GetTargetFor(s.Path)
	seedCfg, err := s.GetConfig()
	if err != nil {
		return nil, err
	}

	// Override dataset from config
	if seedCfg.Schema != "" {
		target.DataSet = seedCfg.Schema
	}
	return target, nil
}

func (s *SeedFile) GetConfig() (*config.SeedConfig, error) {
	configKey := strings.TrimSuffix(s.Path, ".csv")
	return config.GlobalCfg.GetSeedConfig(configKey), nil
}

// ReadColumns reads columns (first row) from CSV file.
func (s *SeedFile) ReadColumns() error {
	f, err := os.Open(s.Path)
	if err != nil {
		return err
	}
	defer f.Close()
	r := csv.NewReader(f)

	headings, err := r.Read()
	if err == io.EOF {
		return fmt.Errorf("Seed file %s has less than one row", s.Path)
	}
	s.Columns = headings

	return s.readColumnTypes()
}

func (s *SeedFile) readColumnTypes() error {
	cfg, err := s.GetConfig()
	if err != nil {
		return err
	}

	if cfg.ColumnTypes == nil {
		// Not specified (use auto detect)
		return nil
	}

	for _, column := range s.Columns {
		colType, ok := cfg.ColumnTypes[column]
		if !ok || colType == "" {
			colType = "string" // default to string
		}
		if s.ColumnTypes == nil {
			s.ColumnTypes = make(map[string]string)
		}
		s.ColumnTypes[column] = colType
	}
	return nil
}

func (s *SeedFile) HasSchema() bool {
	return s.ColumnTypes != nil
}
