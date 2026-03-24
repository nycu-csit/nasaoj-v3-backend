package helper

import "github.com/spf13/cobra"

var RootCmd = &cobra.Command{
	Use:   "helper",
	Short: "Helper commands for container execution",
}

func init() {
	RootCmd.AddCommand(execCmd)
}
