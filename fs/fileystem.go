package fs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FileSystem struct {
	files       []*File
	macroLookup map[string]*File
	modelLookup map[string]*File
}

func ReadFileSystem() (*FileSystem, error) {
	fs := &FileSystem{
		files:       make([]*File, 0),
		macroLookup: make(map[string]*File),
		modelLookup: make(map[string]*File),
	}

	if err := fs.scanDirectory("./macros/", MacroFile); err != nil {
		return nil, err
	}

	if err := fs.scanDirectory("./models/", ModelFile); err != nil {
		return nil, err
	}

	fmt.Printf("🔎 Found %d models, %d macros\n", len(fs.files)-len(fs.macroLookup), len(fs.macroLookup))

	return fs, nil
}

func (fs *FileSystem) scanDirectory(path string, fileType FileType) error {
	return filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		// If we've encountered an error walking this path, let's return now
		if err != nil {
			return err
		}

		// We don't care about directories
		if info.IsDir() {
			return nil
		}

		// We don't care about files which are not SQL
		if filepath.Ext(path) != ".sql" {
			return nil
		}

		file := newFile(path, info, fileType)
		fs.files = append(fs.files, file)

		// For models we want to be able to look them up by partial file name
		switch fileType {
		case MacroFile:
			if err := fs.mapMacroLookupOptions(file); err != nil {
				return err
			}

		case ModelFile:
			if err := fs.mapModelLookupOptions(file); err != nil {
				return err
			}
		}

		return nil
	})
}

// Maps macros into our lookup options
func (fs *FileSystem) mapMacroLookupOptions(file *File) error {
	path := strings.TrimSuffix(filepath.Base(file.Path), ".sql")

	// Add the base path
	if _, found := fs.macroLookup[path]; found {
		return errors.New("macro " + path + " already in lookup")
	}
	fs.macroLookup[path] = file

	return nil
}

// Map all the ways models can be referenced
//
// Mapping all the possible ways we could try
// and look up the file by partial paths
func (fs *FileSystem) mapModelLookupOptions(file *File) error {
	path := strings.TrimSuffix(file.Path, ".sql")

	// Add the base path
	if _, found := fs.modelLookup[path]; found {
		return errors.New("model " + path + " already in lookup")
	}
	fs.modelLookup[path] = file

	// So we can lookup by "model/foo/bar/x" or "foo/bar/x" or "bar/x" as well, let's cache those now
	folders := strings.Split(filepath.Dir(path), string(os.PathSeparator))
	for _, folder := range folders {
		path = strings.TrimPrefix(path, folder+string(os.PathSeparator))

		if _, found := fs.modelLookup[path]; found {
			return errors.New("model " + path + " already in lookup")
		}
		fs.modelLookup[path] = file
	}

	return nil
}

func (fs *FileSystem) NumberFiles() int {
	return len(fs.files)
}

// Returns a model by name or nil if the model is not found
func (fs *FileSystem) Model(name string) *File {
	return fs.modelLookup[name]
}

// Returns a list of all the files
func (fs *FileSystem) Models() []*File {
	models := make([]*File, 0, len(fs.files)-len(fs.macroLookup))

	for _, file := range fs.files {
		if file.Type == ModelFile {
			models = append(models, file)
		}
	}

	return models
}

// Returns a macro by name
func (fs *FileSystem) Macro(name string) *File {
	return fs.macroLookup[name]
}

// Returns a list of macros
func (fs *FileSystem) Macros() []*File {
	macros := make([]*File, 0, len(fs.macroLookup))
	for _, macro := range fs.macroLookup {
		macros = append(macros, macro)
	}

	return macros
}

func (fs *FileSystem) AllFiles() []*File {
	return fs.files
}
