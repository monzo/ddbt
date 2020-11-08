package fs

import (
	"errors"
	"fmt"
	"io/ioutil"
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

func (s *SchemaFile) Parse(fs *FileSystem) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Read and parse the schema file
	bytes, err := ioutil.ReadFile(s.Path)
	if err != nil {
		return err
	}

	err = s.Properties.Unmarshal(bytes)
	if err != nil {
		return err
	}

	// Now attach it to the various models it references
	for _, modelSchema := range s.Properties.Models {
		model := fs.Model(modelSchema.Name)
		if model == nil {
			model = fs.Test(modelSchema.Name)
		}

		if model == nil {
			return errors.New(fmt.Sprintf("Unable to apply model schema; model %s not found", modelSchema.Name))
		}

		model.Schema = modelSchema
	}

	// Add snapshot records
	for _, modelSchema := range s.Properties.Snapshots {
		model := fs.Model(modelSchema.Name)
		if model == nil {
			return errors.New(fmt.Sprintf("Unable to apply snapshot schema; %s not found", modelSchema.Name))
		}

		model.Schema = modelSchema
	}

	return nil
}
