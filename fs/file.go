package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"ddbt/compilerInterface"
	"ddbt/jinja/ast"
)

type FileType string

const (
	UnknownFile FileType = "UNKNOWN"
	ModelFile            = "model"
	MacroFile            = "macro"
	TestFile             = "test"
)

type File struct {
	Type FileType
	Name string
	Path string

	Mutex      sync.Mutex
	SyntaxTree ast.AST

	PrereadFileContents string // Used for testing

	configMutex sync.RWMutex
	config      map[string]*compilerInterface.Value
}

func newFile(path string, file os.FileInfo, fileType FileType) *File {
	return &File{
		Type: fileType,
		Name: strings.TrimSuffix(filepath.Base(path), ".sql"),
		Path: path,

		config: make(map[string]*compilerInterface.Value),
	}
}

func (f *File) SetConfig(name string, value *compilerInterface.Value) {
	f.configMutex.Lock()
	defer f.configMutex.Unlock()
}

func (f *File) ConfigObject() *compilerInterface.Value {
	configObjForFile := compilerInterface.NewMap(map[string]*compilerInterface.Value{
		"get": compilerInterface.NewFunction(func(ec compilerInterface.ExecutionContext, caller compilerInterface.AST, args compilerInterface.Arguments) (*compilerInterface.Value, error) {
			f.configMutex.RLock()
			defer f.configMutex.RUnlock()

			if len(args) < 1 {
				return nil, ec.ErrorAt(caller, "config.get requires at least 1 argument")
			}

			value, found := f.config[args[0].Value.AsStringValue()]
			if !found {
				if len(args) > 1 {
					return args[1].Value, nil
				} else {
					return compilerInterface.NewUndefined(), nil
				}
			}
			return value, nil
		}),

		"require": compilerInterface.NewFunction(func(ec compilerInterface.ExecutionContext, caller compilerInterface.AST, args compilerInterface.Arguments) (*compilerInterface.Value, error) {
			f.configMutex.RLock()
			defer f.configMutex.RUnlock()

			if len(args) != 1 {
				return nil, ec.ErrorAt(caller, "config.require requires 1 argument")
			}

			value, found := f.config[args[0].Value.AsStringValue()]
			if !found {
				return nil, ec.ErrorAt(caller, fmt.Sprintf("%s required but was not set", args[0].Value.AsStringValue()))
			}
			return value, nil
		}),
	})

	configObjForFile.Function = func(ec compilerInterface.ExecutionContext, caller compilerInterface.AST, args compilerInterface.Arguments) (*compilerInterface.Value, error) {
		for _, arg := range args {
			if arg.Name == "" {
				return nil, ec.ErrorAt(caller, "config argument missing name")
			}

			f.SetConfig(arg.Name, arg.Value)
		}
		return compilerInterface.NewUndefined(), nil
	}

	return configObjForFile
}
