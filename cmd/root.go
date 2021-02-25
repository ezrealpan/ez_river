package cmd

import (
	river "ezreal.com.cn/ez_river/cmd/river"
	"github.com/spf13/cobra"
)

var (
	// Used for flags.
	rootCmd = &cobra.Command{
		Use:   "root",
		Short: "A generator for Cobra based Applications",
		Long: `Cobra is a CLI library for Go that empowers applications.
				This application is a tool to generate the needed files
				to quickly create a Cobra application.
				`,
	}
)

// Execute executes the root command.
func Execute() error {
	//NewPipCmd
	rootCmd.AddCommand(river.NewRiverCmd())

	return rootCmd.Execute()
}
