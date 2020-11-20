package cmd

import (
	"context"
	"ddbt/bigquery"
	"ddbt/fs"
	"ddbt/utils"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(seedCommand)
}

var seedCommand = &cobra.Command{
	Use:   "seed",
	Short: "Load data in the data warehouse with seed files",
	Run: func(cmd *cobra.Command, args []string) {
		fileSystem, err := fs.ReadFileSystem(os.Stdout)
		if err != nil {
			fmt.Printf("❌ Unable to read filesystem: %s\n", err)
			os.Exit(1)
		}

		if err := loadSeeds(fileSystem); err != nil {
			fmt.Printf("❌ %s\n", err)
			os.Exit(1)
		}
	},
}

func loadSeeds(fileSystem *fs.FileSystem) error {
	seeds := fileSystem.Seeds()
	pb := utils.NewProgressBar("🌱 Loading Seeds", len(seeds))
	defer pb.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	return fs.ProcessSeeds(
		seeds,
		func(seed *fs.SeedFile) error {
			if err := bigquery.LoadSeedFile(ctx, seed); err != nil {
				return err
			}

			pb.Increment()
			return nil
		},
		nil,
	)

}
