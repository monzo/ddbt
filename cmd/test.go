package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"

	"ddbt/bigquery"
	"ddbt/compiler"
	"ddbt/fs"
	"ddbt/utils"
)

func init() {
	rootCmd.AddCommand(testCmd)
	addModelsFlag(testCmd)
}

var testCmd = &cobra.Command{
	Use:     "test",
	Short:   "Tests the DAG",
	Long:    "Will execute any tests which reference models in the target DAG",
	Example: "ddbt test -m +my_model",
	Run: func(cmd *cobra.Command, args []string) {
		fileSystem, globalContext := compileAllModels()

		// If we've been given a model to run, run it
		graph := buildGraph(fileSystem, ModelFilter)

		// Add all tests which reference the graph
		tests := graph.AddReferencingTests()

		if executeTests(tests, globalContext, graph) {
			os.Exit(2) // Exit with a test error
		}
	},
}

func executeTests(tests []*fs.File, globalContext *compiler.GlobalContext, graph *fs.Graph) bool {
	pb := utils.NewProgressBar("🔬 Running Tests", len(tests))

	_, cancel := context.WithCancel(context.Background())

	var m sync.Mutex
	widestTestName := 0
	type testResult struct {
		file  *fs.File
		name  string
		rows  uint64
		err   error
		query string
	}
	testResults := make(map[*fs.File]testResult)

	_ = fs.ProcessFiles(
		tests,
		func(file *fs.File) error {
			if file.IsDynamicSQL() {
				if err := compiler.CompileModel(file, globalContext, true); err != nil {
					pb.Stop()
					fmt.Printf("❌ %s\n", err)
					cancel()
					os.Exit(1)
				}
			}

			query := bigquery.BuildQuery(file)

			if strings.TrimSpace(query) != "" {
				target, err := file.GetTarget()
				if err != nil {
					pb.Stop()
					fmt.Printf("❌ Unable to get target for %s: %s\n", file.Name, err)
					cancel()
					os.Exit(1)
				}

				rows, err := bigquery.NumberRows(query, target)

				m.Lock()
				testResults[file] = testResult{
					file:  file,
					name:  file.Name,
					rows:  rows,
					err:   err,
					query: query,
				}

				if len(file.Name) > widestTestName {
					widestTestName = len(file.Name)
				}
				m.Unlock()
			}

			pb.Increment()

			return nil
		},
		pb,
	)

	pb.Stop()

	var firstError *testResult

	fmt.Printf("\nTest Results:\n")
	for test, results := range testResults {
		results := results

		// Force this test to be-rerun in future watch loops
		graph.UnmarkFileAsRun(results.file)

		var statusText string
		var statusEmoji rune

		switch {
		case results.err == context.Canceled:
			statusText = "Cancelled"
			statusEmoji = '🚧'

		case results.err != nil:
			statusText = fmt.Sprintf("Error: %s", results.err)
			statusEmoji = '🔴'

		case results.rows > 0:
			statusText = fmt.Sprintf("%d Failures", results.rows)
			statusEmoji = '❌'

		default:
			statusText = "Success"
			statusEmoji = '✅'
		}

		if firstError == nil && statusEmoji != '✅' {
			firstError = &results
		}

		fmt.Printf(
			"   %c  %s %s %s\n",
			statusEmoji,
			test.Name,
			strings.Repeat(".", widestTestName-len(test.Name)+3),
			statusText,
		)
	}

	if firstError != nil {
		if err := clipboard.WriteAll(firstError.query); err != nil {
			fmt.Printf("   Unable to copy query to clipboard: %s\n", err)
		} else {
			fmt.Printf("📎 Test Query for %s has been copied into your clipboard\n\n", firstError.name)
		}

		return true
	}

	return false
}
