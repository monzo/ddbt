package fs

import (
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"
)

type DocFile struct {
	Name     string
	Path     string
	Contents string

	mutex sync.Mutex
}

func newDocFile(path string) *DocFile {
	return &DocFile{
		Name:     strings.TrimSuffix(filepath.Base(path), ".md"),
		Path:     path,
		Contents: "",
	}
}

func (d *DocFile) GetName() string {
	return d.Name
}

func (d *DocFile) Parse(fs *FileSystem) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	// Read and parse the schema file
	bytes, err := ioutil.ReadFile(d.Path)
	if err != nil {
		return err
	}

	d.Contents = string(bytes)
	return nil
}
