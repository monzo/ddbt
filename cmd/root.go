package cmd

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/spf13/cobra"

	"ddbt/bigquery"
	"ddbt/compiler"
	"ddbt/config"
	"ddbt/utils"
)

var rootCmd = &cobra.Command{
	Use:     "ddbt",
	Short:   "Dom's Data Build tool is very fast version of DBT",
	Long:    "DDBT is an experimental drop in replacement for DBT which aims to be much faster at building the DAG for projects with large numbers of models",
	Version: utils.DdbtVersion,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Do not run init if we're running info commands which aren't actually going to execute operate on a project
		if cmd != versionCmd && cmd.Name() != "help" {
			initDDBT()
		}
	},
}

var (
	targetProfile   string
	upstreamProfile string
	threads         int
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&targetProfile, "target", "t", "", "Which target profile to use")
	rootCmd.PersistentFlags().StringVarP(&upstreamProfile, "upstream", "u", "", "Which target profile to use when reading data outside the current DAG")
	rootCmd.PersistentFlags().IntVar(&threads, "threads", 0, "How many threads to execute with")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initDDBT() {
	// If you happen to be one folder up from the DBT project, we'll cd in there for you to be nice :)
	cdIntoDBTFolder()

	// Read the project config
	cfg, err := config.Read(targetProfile, upstreamProfile, threads, compiler.CompileStringWithCache)
	if err != nil {
		fmt.Printf("❌ Unable to load config: %s\n", err)
		os.Exit(1)
	}

	// Init our connection to BigQuery
	if err := bigquery.Init(cfg); err != nil {
		fmt.Printf("❌ Unable to init BigQuery: %s\n", err)
		os.Exit(1)
	}
}

func cdIntoDBTFolder() {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	if path.Base(wd) != "dbt" {
		if stat, err := os.Stat(filepath.Join(wd, "dbt")); !os.IsNotExist(err) && stat.IsDir() {
			err = os.Chdir(filepath.Join(wd, "dbt"))
			if err != nil {
				panic(err)
			}
		}
	}
}
