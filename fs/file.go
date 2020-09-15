package fs

import (
	"errors"
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

	Mutex            sync.Mutex
	SyntaxTree       ast.AST
	CompiledContents string

	PrereadFileContents string // Used for testing

	configMutex sync.RWMutex
	config      map[string]*compilerInterface.Value

	// Graph tracking
	upstreams   map[*File]struct{}
	downstreams map[*File]struct{}
}

func newFile(path string, file os.FileInfo, fileType FileType) *File {
	return &File{
		Type: fileType,
		Name: strings.TrimSuffix(filepath.Base(path), ".sql"),
		Path: path,

		config:      make(map[string]*compilerInterface.Value),
		upstreams:   make(map[*File]struct{}),
		downstreams: make(map[*File]struct{}),
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

// Record an file as upstream to this file
func (f *File) RecordDependencyOn(upstream *File) {
	// No need to record a dependency on ourselves
	if f == upstream {
		return
	}

	f.Mutex.Lock()
	f.upstreams[upstream] = struct{}{}
	f.Mutex.Unlock()

	upstream.Mutex.Lock()
	upstream.downstreams[f] = struct{}{}
	upstream.Mutex.Unlock()
}

func (f *File) BuildUpstreamDag(depth int, depthMap map[*File]int, downstreams map[*File]struct{}) (largestDepth int, err error) {
	if _, found := downstreams[f]; found {
		return 0, errors.New(fmt.Sprintf("circular dependency detected in %s", f.Path))
	}

	if currentDepth, found := depthMap[f]; found && currentDepth >= depth {
		// We've already visited this file and it's already recorded at a greater depth means
		// we can just return now, otherwise we need to re-record this file at a greater depth
		return currentDepth, nil
	}

	depthMap[f] = depth
	largestDepth = depth

	// Copy the downstreams of this model
	newDownstreams := make(map[*File]struct{})
	for key := range downstreams {
		newDownstreams[key] = struct{}{}
	}

	// add this model to the new downstreams
	newDownstreams[f] = struct{}{}

	f.Mutex.Lock()
	defer f.Mutex.Unlock()
	for upstream := range f.upstreams {
		foundDepth, err := upstream.BuildUpstreamDag(depth+1, depthMap, newDownstreams)
		if err != nil {
			return 0, err
		}

		if foundDepth > largestDepth {
			largestDepth = foundDepth
		}
	}

	return largestDepth, nil
}
