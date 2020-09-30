package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"ddbt/utils"
	"ddbt/watcher"
)

func init() {
	rootCmd.AddCommand(watchCmd)
	watchCmd.Flags().StringVarP(&ModelFilter, "models", "m", "", "Select which model(s) to watch")
}

var watchCmd = &cobra.Command{
	Use:     "watch",
	Short:   "Automagically runs your DAG & tests whenever you change them",
	Long:    "This mode will run all the models in your DAG and any related tests. Whenever you change a file used in the DAG that part of the DAG will be automatically re-run and re-tested.",
	Example: "ddbt watch -m +my_model",
	Run: func(cmd *cobra.Command, args []string) {
		// Do the initial build of the models and then add the tests
		fileSystem, _ := compileAllModels()
		graph := buildGraph(fileSystem, ModelFilter)
		graph.AddReferencingTests()

		watchLoop()
	},
}

func watchLoop() {
	utils.ClearTerminal()

	// Start watching the file system
	watch, err := watcher.NewWatcher()
	if err != nil {
		fmt.Printf("❌ Unable to start watching the file system for changes: %s\n", err)
		os.Exit(1)
	}
	defer watch.Close()

	addWatch(watch, "./macros")
	addWatch(watch, "./models")
	addWatch(watch, "./tests")

	for range watch.EventsReady {
		// Note; we batch events so if the DAG takes 5 minutes and the user changes 20 files
		// in that time, we don't loop 20 more times - instead we pick up all 20 changes at once
		// and run the DAG 1 more time
		batch := watch.GetEventsBatch()

		fmt.Println("Events:")
		for _, event := range batch.Events() {
			fmt.Printf("\t%+v\n", event)
		}
	}
}

func addWatch(watcher *watcher.Watcher, folder string) {
	if err := watcher.RecursivelyWatch(folder); err != nil {
		fmt.Printf("❌ Unable to start subdirectories of %s: %s\n", folder, err)
		os.Exit(1)
	}
}
