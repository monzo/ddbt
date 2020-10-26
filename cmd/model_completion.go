package cmd

import (
	"fmt"
	"io/ioutil"
	"strings"
	"sync"

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

// matchModel returns a list of models with the given prefix.
// If prefix is empty, it returns all models.
func matchModel(prefix string) []string {
	readFSOnce.Do(func() {
		// Read filesystem w/o output as it'll interfere with autocompletion
		fileSystem, err := fs.ReadFileSystem(ioutil.Discard)
		if err != nil {
			cobra.CompError(fmt.Sprintf("❌ Unable to read filesystem: %s\n", err))
		}
		cachedFS = fileSystem
	})
	if cachedFS != nil {
		matched := make([]string, 0, len(cachedFS.Models()))
		for _, m := range cachedFS.Models() {
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
