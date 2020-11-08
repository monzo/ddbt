package fs

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type FileSystem struct {
	files       map[string]*File       // path -> File
	macroLookup map[string]*File       // macro name -> File
	modelLookup map[string]*File       // model lookup name -> File
	schemas     map[string]*SchemaFile // schema files
	tests       []*File                // Tests
}

func ReadFileSystem(msgWriter io.Writer) (*FileSystem, error) {
	fs := &FileSystem{
		files:       make(map[string]*File),
		macroLookup: make(map[string]*File),
		modelLookup: make(map[string]*File),
		schemas:     make(map[string]*SchemaFile),
		tests:       make([]*File, 0),
	}

	// FIXME: disabled for a bit
	if err := fs.scanDBTModuleMacros(); err != nil {
		return nil, err
	}

	if err := fs.scanDirectory("./macros/", MacroFile); err != nil {
		return nil, err
	}

	if err := fs.scanDirectory("./models/", ModelFile); err != nil {
		return nil, err
	}

	if err := fs.scanDirectory("./tests/", TestFile); err != nil {
		return nil, err
	}

	_, _ = fmt.Fprintf(msgWriter, "🔎 Found %d models, %d macros, %d tests, %d schema files\n", len(fs.files)-len(fs.macroLookup)-len(fs.tests), len(fs.macroLookup), len(fs.tests), len(fs.schemas))

	return fs, nil
}

// Create a test file system with mock files
func InMemoryFileSystem(models map[string]string) (*FileSystem, error) {
	fs := &FileSystem{
		files:       make(map[string]*File, 0),
		macroLookup: make(map[string]*File),
		modelLookup: make(map[string]*File),
		schemas:     make(map[string]*SchemaFile),
		tests:       make([]*File, 0),
	}

	for filePath, contents := range models {
		filePath = filepath.Clean(filePath)

		file := newFile(filePath, ModelFile)
		file.PrereadFileContents = contents

		fs.files[filePath] = file

		if err := fs.mapModelLookupOptions(file); err != nil {
			return nil, err
		}
	}

	return fs, nil
}

// Scan any macros in our dbt modules folder
func (fs *FileSystem) scanDBTModuleMacros() error {
	files, err := ioutil.ReadDir("./dbt_modules")
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		} else {
			return err
		}
	}

	for _, f := range files {
		if f.IsDir() {
			if err := fs.scanDirectory("./dbt_modules/"+f.Name()+"/macros", MacroFile); err != nil {
				return err
			}
		}
	}

	return nil
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

		path = filepath.Clean(path)

		switch filepath.Ext(path) {
		case ".sql":
			return fs.recordSQLFile(path, fileType)

		case ".yml":
			return fs.recordSchemaFile(path)
		}

		return nil
	})
}

func (fs *FileSystem) recordSQLFile(path string, fileType FileType) error {
	file := newFile(path, fileType)
	fs.files[path] = file

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

	case TestFile:
		if err := fs.mapTestLookupOptions(file); err != nil {
			return err
		}
	}

	return nil
}

func (fs *FileSystem) recordSchemaFile(path string) error {
	fs.schemas[path] = newSchemaFile(path)

	return nil
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

// Map tests into our lookup options
func (fs *FileSystem) mapTestLookupOptions(file *File) error {
	fs.tests = append(fs.tests, file)
	return nil
}

func (fs *FileSystem) NumberFiles() int {
	return len(fs.files)
}

func (fs *FileSystem) NumberSchemas() int {
	return len(fs.schemas)
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

// Returns a list of tests
func (fs *FileSystem) Tests() []*File {
	tests := make([]*File, 0, len(fs.tests))
	for _, macro := range fs.tests {
		tests = append(tests, macro)
	}

	return tests
}

func (fs *FileSystem) AllFiles() []*File {
	files := make([]*File, 0, len(fs.files))

	for _, file := range fs.files {
		files = append(files, file)
	}

	return files
}

func (fs *FileSystem) File(path string, info os.FileInfo) (*File, error) {
	file, found := fs.files[path]

	if !found {
		if filepath.Ext(path) != ".sql" {
			return nil, nil
		}

		fileType := UnknownFile
		if strings.HasPrefix(path, "macros") {
			fileType = MacroFile
		} else if strings.HasPrefix(path, "models") {
			fileType = ModelFile
		} else if strings.HasPrefix(path, "tests") {
			fileType = TestFile
		}

		if err := fs.recordSQLFile(path, fileType); err != nil {
			return nil, err
		}

		return fs.files[path], nil
	}

	return file, nil
}

func (fs *FileSystem) AllSchemas() []*SchemaFile {
	schemas := make([]*SchemaFile, 0, len(fs.schemas))

	for _, schema := range fs.schemas {
		schemas = append(schemas, schema)
	}

	return schemas
}
