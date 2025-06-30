package utils

import "github.com/spf13/cobra"

var RootCmd = &cobra.Command{
	Use:   "utils",
	Short: "NASAOJ utils",
}

func init() {
	RootCmd.AddCommand(importUserCmd)
	RootCmd.AddCommand(exportScoresCmd)
}
