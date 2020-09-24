package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"ddbt/bigquery"
	"ddbt/compiler"
	"ddbt/config"
)

var rootCmd = &cobra.Command{
	Use:   "ddbt",
	Short: "Dom's Data Build tool is very fast version of DBT",
	Long:  "DDBT is an experimental drop in replacement for DBT which aims to be much faster at building the DAG for projects with large numbers of models",
}

var (
	targetProfile   string
	upstreamProfile string
	threads         int
)

func init() {
	cobra.OnInitialize(initDDBT)

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
