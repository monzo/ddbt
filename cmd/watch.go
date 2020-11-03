package cmd

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"

	"ddbt/bigquery"
	"ddbt/compiler"
	"ddbt/config"
	"ddbt/fs"
	"ddbt/utils"
	"ddbt/watcher"
)

var skipInitialBuild = false

func init() {
	rootCmd.AddCommand(watchCmd)
	addModelsFlag(watchCmd)

	watchCmd.Flags().BoolVarP(&skipInitialBuild, "skip-run", "s", false, "Skip the initial execution of the DAG and go straight into watch mode")
}

var watchCmd = &cobra.Command{
	Use:     "watch",
	Short:   "Automagically runs your DAG & tests whenever you change them",
	Long:    "This mode will run all the models in your DAG and any related tests. Whenever you change a file used in the DAG that part of the DAG will be automatically re-run and re-tested.",
	Example: "ddbt watch -m +my_model",
	Run: func(cmd *cobra.Command, args []string) {
		// Do the initial build of the models and then add the tests
		fileSystem, gc := compileAllModels()
		graph := buildGraph(fileSystem, ModelFilter)
		testsToRun := graph.AddReferencingTests()

		if !skipInitialBuild {
			_ = executeGraph(graph, gc)
		}

		// All tests are always executed first, to get initial test state
		executeTests(testsToRun, gc, graph)

		watchLoop(fileSystem, graph, gc)
	},
}

func watchLoop(fileSystem *fs.FileSystem, graph *fs.Graph, gc *compiler.GlobalContext) {
	// At this point we only want to re-run things which changed after entering watch mode
	graph.MarkGraphAsFullyRun()

	fmt.Printf("🕵️Entering watch mode")

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

	for {
		fmt.Printf("\n⏳  Waiting for you to save changes to the DAG...\n")
		<-watch.EventsReady
		utils.ClearTerminal()

		// Note; we batch events so if the DAG takes 5 minutes and the user changes 20 files
		// in that time, we don't loop 20 more times - instead we pick up all 20 changes at once
		// and run the DAG 1 more time
		batch := watch.GetEventsBatch()

		// Handle all the events, and figure out what files we need to reread and parse
		filesToReParse := make([]*fs.File, 0)
		for _, event := range batch.Events() {
			filesToReParse = handleEvent(fileSystem, graph, gc, event, filesToReParse)
		}

		if len(filesToReParse) == 0 {
			// No file needs re-parsing, great we can skip this set of events
			continue
		}

		// Now reparse any changed files
		if err := handleReparses(graph, filesToReParse); err != nil {
			continue
		}

		// And now recompile anything which needs recompiling
		if err := handleRecompiles(fileSystem, gc); err != nil {
			continue
		}

		// TODO: detect and handle new DAG nodes and edges after the recompile

		// Now re-run the models in the which changed
		testsToRun, err := handleRerun(graph, gc)
		if err != nil {
			continue
		}

		// Now re-run any tests
		if len(testsToRun) > 0 {
			executeTests(testsToRun, gc, graph)
		}
	}
}

func handleEvent(fileSystem *fs.FileSystem, graph *fs.Graph, gc *compiler.GlobalContext, event watcher.Event, filesToRecompile []*fs.File) []*fs.File {
	// Get the file (this will also create it for new files)
	file, err := fileSystem.File(event.Path, event.Info)
	if err != nil {
		fmt.Printf("⚠️ Unable to get %s out of the in-memory filesystem: %s\n", event.Path, err)
	} else if file != nil {
		// First invalid all down streams within the DAG (and if this is in the DAG too
		invalidateDownstreams(file, graph)

		file.SyntaxTree = nil // Invalidate the syntax tree of this file
		if file.Type == fs.MacroFile {
			gc.UnregisterMacrosInFile(file)
		}

		if event.EventType == watcher.DELETED {
			// TODO: handle deleted files
		} else {
			filesToRecompile = append(filesToRecompile, file)
		}
	}

	return filesToRecompile
}

func handleReparses(graph *fs.Graph, filesToReParse []*fs.File) error {
	numFiles := len(filesToReParse)
	switch {
	case numFiles == 1:
		fmt.Printf("🔎 %s needs to be recompiled", filesToReParse[0].Name)
	case numFiles < 5:
		fmt.Printf("🔎 ")
		for i, file := range filesToReParse {
			if i > 0 {
				fmt.Printf(", ")
			}

			fmt.Printf(file.Name)
		}
		fmt.Printf(" all need recompling")
	default:
		fmt.Printf("🔎 %d files need recompiling", numFiles)

	}
	fmt.Printf(", %d nodes in the DAG need to be re-run\n", graph.NumberNodesNeedRerunning())

	pb := utils.NewProgressBar("📜 Reading & Parsing Files", numFiles)
	defer pb.Stop()

	return fs.ProcessFiles(filesToReParse, func(file *fs.File) error {
		if err := compiler.ParseFile(file); err != nil {
			pb.Stop()
			fmt.Printf("⚠️ Unable to parse %s %s: %s\n", file.Type, file.Name, err)
			return err
		}

		pb.Increment()
		return nil
	}, nil)
}

func handleRecompiles(fileSystem *fs.FileSystem, gc *compiler.GlobalContext) error {
	macrosToRecompile := make([]*fs.File, 0)
	modelsToRecompile := make([]*fs.File, 0)
	testsToRecompile := make([]*fs.File, 0)

	for _, file := range fileSystem.AllFiles() {
		if file.NeedsRecompile {
			switch file.Type {
			case fs.MacroFile:
				macrosToRecompile = append(macrosToRecompile, file)
			case fs.ModelFile:
				modelsToRecompile = append(modelsToRecompile, file)
			case fs.TestFile:
				testsToRecompile = append(testsToRecompile, file)
			default:
				fmt.Printf("⚠️ File %s has an unknown file type: %s\n", file.Name, file.Type)
			}
		}
	}

	compile := func(name string, list []*fs.File) error {
		if len(list) == 0 {
			return nil
		}

		pb := utils.NewProgressBar(name, len(list))
		defer pb.Stop()

		return fs.ProcessFiles(list, func(file *fs.File) error {
			if err := compiler.CompileModel(file, gc, false); err != nil {
				pb.Stop()
				fmt.Printf("⚠️ Unable to compile %s %s: %s\n", file.Type, file.Name, err)
				return err
			}

			file.NeedsRecompile = false

			pb.Increment()

			return nil
		}, nil)
	}

	if err := compile("📚 Compiling Macros", macrosToRecompile); err != nil {
		return err
	}
	if err := compile("📝 Compiling Models", modelsToRecompile); err != nil {
		return err
	}
	if err := compile("🧪 Compiling Tests", testsToRecompile); err != nil {
		return err
	}

	return nil
}

func handleRerun(graph *fs.Graph, gc *compiler.GlobalContext) ([]*fs.File, error) {
	pb := utils.NewProgressBar("🚀 Executing impacted models", graph.NumberNodesNeedRerunning())
	defer pb.Stop()

	var testMutex sync.Mutex
	foundTests := make([]*fs.File, 0)

	ctx, cancel := context.WithCancel(context.Background())

	err := graph.Execute(func(file *fs.File) error {
		if file.Type == fs.TestFile {
			testMutex.Lock()
			foundTests = append(foundTests, file)
			testMutex.Unlock()
		} else if file.Type == fs.ModelFile && file.GetMaterialization() != "ephemeral" {
			if file.IsDynamicSQL() || upstreamProfile != "" {
				if err := compiler.CompileModel(file, gc, true); err != nil {
					pb.Stop()
					fmt.Printf("⚠️ Unable to recompile dynamic model %s: %s\n", file.Name, err)
					cancel()
					return err
				}
			}

			if queryStr, err := bigquery.Run(ctx, file); err != nil {
				pb.Stop()

				if err != context.Canceled {
					fmt.Printf("⚠️ %s\n", err)

					if err := clipboard.WriteAll(queryStr); err != nil {
						fmt.Printf("   Unable to copy query to clipboard: %s\n", err)
					} else {
						fmt.Printf("📎 Query has been copied into your clipboard\n\n")
					}
				}

				cancel()
				return err
			}
		}

		pb.Increment()

		return nil
	}, config.GlobalCfg.Target.Threads, pb)

	return foundTests, err
}

func invalidateDownstreams(file *fs.File, graph *fs.Graph) {
	type visit struct {
		file                   *fs.File
		removeCompiledContents bool
	}
	toVisit := []visit{{file, true}}
	queued := map[*fs.File]struct{}{file: {}}

	for len(toVisit) > 0 {
		node := toVisit[0]
		toVisit = toVisit[1:]

		// Remove the compiled contents
		if node.removeCompiledContents {
			node.file.CompiledContents = ""
			node.file.NeedsRecompile = true
		}

		// If it's in the graph, remove it's
		graph.UnmarkFileAsRun(node.file)

		// If this is a macro, or it's ephemeral downstreams need to have their compiled contents reset
		downstreamRecompile := node.file.Type == fs.MacroFile || node.file.GetMaterialization() == "ephemeral"

		for _, downstream := range node.file.Downstreams() {
			if _, found := queued[downstream]; !found {
				queued[downstream] = struct{}{}
				toVisit = append(toVisit, visit{downstream, downstreamRecompile})
			}
		}
	}
}

func addWatch(watcher *watcher.Watcher, folder string) {
	if err := watcher.RecursivelyWatch(folder); err != nil {
		fmt.Printf("❌ Unable to start subdirectories of %s: %s\n", folder, err)
		os.Exit(1)
	}
}
