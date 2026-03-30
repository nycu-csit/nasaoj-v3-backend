package tests

import (
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "test",
	Short: "Test problems or tools",
}

func init() {
	RootCmd.AddCommand(problemCmd)
	RootCmd.AddCommand(toolCmd)
}
