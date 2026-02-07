package disable

import "github.com/spf13/cobra"

var RootCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable resources",
}

func init() {
	RootCmd.AddCommand(problemCmd)
	RootCmd.AddCommand(toolCmd)
	RootCmd.AddCommand(tokenCmd)
}
