package get

import "github.com/spf13/cobra"

var RootCmd = &cobra.Command{
	Use:   "get",
	Short: "Get resources",
}

func init() {
	RootCmd.AddCommand(problemsCmd)
	RootCmd.AddCommand(toolsCmd)
	RootCmd.AddCommand(userCmd)
	RootCmd.AddCommand(tokensCmd)
	RootCmd.AddCommand(workersCmd)
}
