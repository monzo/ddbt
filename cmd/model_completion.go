package cmd

import (
	"fmt"
	"io/ioutil"
	"sort"
	"strings"
	"sync"

	"ddbt/compiler"
	"ddbt/config"
	"ddbt/fs"

	"github.com/spf13/cobra"
)

var (
	readFSOnce sync.Once
	cachedFS   *fs.FileSystem
)

// completeModelFn is a custom valid argument function for cobra.Command.
func completeModelFn(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		// Only complete the first arg.
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return matchModel(toComplete), cobra.ShellCompDirectiveNoFileComp
}

// completeModelFilterFn is a custom valid argument function for cobra.Command.
// It is a variation of completeModelFn, but supports model filters
// (e.g. specifying upstream with + or with tags)
func completeModelFilterFn(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		// Only complete the first arg.
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	switch {
	case strings.HasPrefix(toComplete, "tag:"): // try to match tag:key=value
		return matchTag(toComplete[4:]), cobra.ShellCompDirectiveNoFileComp
	case strings.HasPrefix(toComplete, "+"): //  try to match +model_name
		models := matchModel(toComplete[1:])
		suggestion := make([]string, 0, len(models))
		for _, m := range models {
			suggestion = append(suggestion, "+"+m)
		}
		return suggestion, cobra.ShellCompDirectiveNoFileComp
	}
	// Normal model matching
	return matchModel(toComplete), cobra.ShellCompDirectiveNoFileComp
}

func getFileSystem() *fs.FileSystem {
	readFSOnce.Do(func() {
		// Read filesystem w/o output as it'll interfere with autocompletion
		fileSystem, err := fs.ReadFileSystem(ioutil.Discard)
		if err != nil {
			cobra.CompError(fmt.Sprintf("❌ Unable to read filesystem: %s\n", err))
		}
		cachedFS = fileSystem
	})
	return cachedFS
}

// matchModel returns a list of models with the given prefix.
// If prefix is empty, it returns all models.
func matchModel(prefix string) []string {
	fileSys := getFileSystem()
	if fileSys != nil {
		matched := make([]string, 0, len(fileSys.Models()))
		for _, m := range fileSys.Models() {
			if strings.HasPrefix(m.Name, prefix) {
				// Include as suggestion:
				//   model_name -- full/path/to/model_name.sql
				matched = append(matched, fmt.Sprintf("%s\t%s", m.Name, m.Path))
			}
		}
		return matched
	}
	return nil // Nothing matched
}

// matchModel returns a list of tags with the given prefix.
// If prefix is empty, it returns all tags.
// A tag is in the format of key=value.
func matchTag(prefix string) []string {
	fileSys := getFileSystem()
	if fileSys != nil {
		tags := getAllTags(fileSys)
		var matched []string
		for tagName, files := range tags {
			if strings.HasPrefix(tagName, prefix) {
				// Include as suggestion:
				//   tag:key=value -- Matches N models
				matched = append(matched, fmt.Sprintf("tag:%s\tMatches %d models", tagName, len(files)))
			}
		}
		// Sort output so consecutive autocompletes suggestions are stable
		sort.Strings(matched)
		return matched
	}
	return nil
}

// getAllTags reads all the tags on models.
func getAllTags(fileSys *fs.FileSystem) map[string][]string {
	if fileSys != nil {
		tags := make(map[string][]string)
		// This is simplified version of compileAllModels() to read all tags
		// from the models.
		for _, f := range fileSys.AllFiles() {
			if err := compiler.ParseFile(f); err != nil {
				cobra.CompError(fmt.Sprintf("❌ Unable to parse file %s: %s\n", f.Path, err))
				continue
			}
		}
		gc, err := compiler.NewGlobalContext(config.GlobalCfg, fileSys)
		if err != nil {
			cobra.CompError(fmt.Sprintf("❌ Unable to create a global context: %s\n", err))
		}

		for _, f := range append(fileSys.Macros(), fileSys.Models()...) {
			if err := compiler.CompileModel(f, gc, false); err != nil {
				cobra.CompError(fmt.Sprintf("❌ Unable to compile file %s: %s\n", f.Path, err))
				continue
			}
			for _, tag := range f.GetTags() {
				tags[tag] = append(tags[tag], f.Name)
			}
		}
		return tags
	}
	return nil
}
