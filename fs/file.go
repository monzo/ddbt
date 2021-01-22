package fs

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"ddbt/compilerInterface"
	"ddbt/config"
	"ddbt/jinja/ast"
	"ddbt/properties"
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

	cfgMutex     sync.Mutex
	FolderConfig config.ModelConfig

	Mutex            sync.Mutex
	Schema           *properties.Model // The model schema (if we have one)
	SyntaxTree       ast.AST
	isDynamicSQL     bool // does this need recompiling as part of the DAG?
	CompiledContents string
	NeedsRecompile   bool // used in watch mode

	PrereadFileContents string // Used for testing

	configMutex sync.RWMutex
	config      map[string]*compilerInterface.Value

	// Graph tracking
	upstreams   map[*File]struct{}
	downstreams map[*File]struct{}

	//ctesMutex     sync.Mutex
	EphemeralCTES map[string]*File
	isInDAG       bool
}

func newFile(path string, fileType FileType) *File {
	return &File{
		Type:         fileType,
		Name:         strings.TrimSuffix(filepath.Base(path), ".sql"),
		Path:         path,
		FolderConfig: config.GetFolderConfig(path),

		config:      make(map[string]*compilerInterface.Value),
		upstreams:   make(map[*File]struct{}),
		downstreams: make(map[*File]struct{}),

		EphemeralCTES: make(map[string]*File),
	}
}

func (f *File) GetName() string {
	return f.Name
}

func (f *File) SetConfig(name string, value *compilerInterface.Value) {
	f.configMutex.Lock()
	defer f.configMutex.Unlock()

	f.config[name] = value
}

func (f *File) GetConfig(name string) *compilerInterface.Value {
	f.configMutex.RLock()
	defer f.configMutex.RUnlock()

	if value, found := f.config[name]; found {
		return value.Unwrap()
	} else {
		// If not overridden pick up from the folder default
		switch name {
		case "enabled":
			return compilerInterface.NewBoolean(f.FolderConfig.Enabled)

		case "tags":
			return compilerInterface.NewStringList(f.FolderConfig.Tags)

		case "pre_hooks":
			return compilerInterface.NewStringList(f.FolderConfig.PreHooks)

		case "post_hooks":
			return compilerInterface.NewStringList(f.FolderConfig.PostHooks)

		case "materialized":
			return compilerInterface.NewString(f.FolderConfig.Materialized)

		default:
			return compilerInterface.NewUndefined()
		}
	}
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

		if materialized := f.GetConfig("materialized"); materialized.Type() == compilerInterface.StringVal && materialized.StringValue != "" {
			f.cfgMutex.Lock()
			f.FolderConfig.Materialized = materialized.StringValue
			f.cfgMutex.Unlock()
		}

		if tagsList := f.GetConfig("tags"); tagsList.Type() == compilerInterface.ListVal && tagsList.ListValue != nil {
			f.cfgMutex.Lock()
			for _, tagVal := range tagsList.ListValue {
				if tagVal.Type() == compilerInterface.StringVal {
					f.FolderConfig.Tags = append(f.FolderConfig.Tags, tagVal.StringValue)
				}
			}
			f.cfgMutex.Unlock()
		}

		if enabledValue := f.GetConfig("enabled"); enabledValue.Type() == compilerInterface.BooleanValue && !enabledValue.BooleanValue {
			// This model is not enable and should return undefined (this stops the execution of the AST for the model)
			return compilerInterface.NewReturnValue(compilerInterface.NewUndefined()), nil
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

// All the downstreams in this file
func (f *File) Downstreams() []*File {
	f.Mutex.Lock()
	defer f.Mutex.Unlock()

	downstreams := make([]*File, 0, len(f.downstreams))
	for downstream := range f.downstreams {
		downstreams = append(downstreams, downstream)
	}

	return downstreams
}

// All the upstreams in this file
func (f *File) Upstreams() []*File {
	f.Mutex.Lock()
	defer f.Mutex.Unlock()

	upstreams := make([]*File, 0, len(f.upstreams))
	for upstream := range f.upstreams {
		upstreams = append(upstreams, upstream)
	}

	return upstreams
}

func (f *File) MaskAsDynamicSQL() {
	f.Mutex.Lock()
	defer f.Mutex.Unlock()
	f.isDynamicSQL = true
}

func (f *File) IsDynamicSQL() bool {
	f.Mutex.Lock()
	defer f.Mutex.Unlock()

	return f.isDynamicSQL
}

func (f *File) GetTarget() (*config.Target, error) {
	target := config.GlobalCfg.GetTargetFor(f.Path)

	// Tests may not define these, so we can pull them from the model they are being tested against
	if f.Type == TestFile && (target.ProjectID == "" || target.DataSet == "") {
		f.Mutex.Lock()
		if len(f.upstreams) > 0 {
			for upstream := range f.upstreams {
				upstreamTarget, err := upstream.GetTarget()
				if err != nil {
					return nil, err
				}

				target = upstreamTarget
				break
			}
		}
		f.Mutex.Unlock()
	}

	// Has the model overridden the execution project it needs to run under?
	if value := f.GetConfig("execution_project"); value.Type() == compilerInterface.StringVal {
		target = target.Copy()
		target.ExecutionProjects = []string{value.StringValue}
	}

	// Has the model overridden the project it writes into?
	if value := f.GetConfig("project"); value.Type() == compilerInterface.StringVal {
		target = target.Copy()

		target.ProjectID = value.StringValue
	}

	// Has the model overridden the dataset it writes into?
	if value := f.GetConfig("schema"); value.Type() == compilerInterface.StringVal {
		target = target.Copy()

		target.DataSet = value.StringValue
	}

	// Through project tag substitution, do we need to replace the project id?
	switch value := f.GetConfig("tags"); value.Type() {
	case compilerInterface.ListVal:
		for _, tagValue := range value.ListValue {
			if tagValue.Type() != compilerInterface.StringVal {
				return nil, errors.New(fmt.Sprintf("model %s has a tag which is not a string in the 'name=value' format: %s", f.Name, tagValue))
			}

			if subs, found := target.ProjectSubstitutions[tagValue.StringValue]; found {
				if sub, found := subs[target.ProjectID]; found {
					target = target.Copy()
					target.ProjectID = sub
				}
			}

			if target.ReadUpstream != nil {
				if subs, found := target.ReadUpstream.ProjectSubstitutions[tagValue.StringValue]; found {
					if sub, found := subs[target.ReadUpstream.ProjectID]; found {
						target = target.Copy()
						target.ReadUpstream.ProjectID = sub
					}
				}
			}
		}

	case compilerInterface.StringVal:
		if subs, found := target.ProjectSubstitutions[value.StringValue]; found {
			if sub, found := subs[target.ProjectID]; found {
				target = target.Copy()
				target.ProjectID = sub
			}
		}

		if target.ReadUpstream != nil {
			if subs, found := target.ReadUpstream.ProjectSubstitutions[value.StringValue]; found {
				if sub, found := subs[target.ReadUpstream.ProjectID]; found {
					target = target.Copy()
					target.ReadUpstream.ProjectID = sub
				}
			}
		}
	}

	return target, nil
}

func (f *File) GetMaterialization() string {
	f.cfgMutex.Lock()
	defer f.cfgMutex.Unlock()

	return f.FolderConfig.Materialized
}

func (f *File) GetTags() []string {
	f.cfgMutex.Lock()
	defer f.cfgMutex.Unlock()

	return f.FolderConfig.Tags
}

func (f *File) MarkAsInDAG() {
	f.cfgMutex.Lock()
	defer f.cfgMutex.Unlock()

	f.isInDAG = true
}

func (f *File) IsInDAG() bool {
	f.cfgMutex.Lock()
	defer f.cfgMutex.Unlock()

	return f.isInDAG
}

func (f *File) HasTag(tag string) bool {
	tags := f.GetTags()

	for _, t := range tags {
		if t == tag {
			return true
		}
	}

	return false
}
