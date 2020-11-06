package fs

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"ddbt/properties"
)

type SchemaFile struct {
	Name       string
	Path       string
	Properties *properties.File

	mutex sync.Mutex
}

func newSchemaFile(path string) *SchemaFile {
	return &SchemaFile{
		Name:       strings.TrimSuffix(filepath.Base(path), ".yml"),
		Path:       path,
		Properties: &properties.File{},
	}
}

func (s *SchemaFile) GetName() string {
	return s.Name
}

func (s *SchemaFile) Parse() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	f, err := os.Open(s.Path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	return s.Properties.Unmarshal(bytes)
}
