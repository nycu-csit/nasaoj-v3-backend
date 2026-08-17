package get

import "github.com/spf13/cobra"

var RootCmd = &cobra.Command{
	Use:   "get",
	Short: "Get resources",
}

func init() {
	RootCmd.AddCommand(problemCmd)
	RootCmd.AddCommand(problemsCmd)
	RootCmd.AddCommand(toolsCmd)
	RootCmd.AddCommand(userCmd)
	RootCmd.AddCommand(usersCmd)
	RootCmd.AddCommand(tokensCmd)
	RootCmd.AddCommand(workersCmd)
	RootCmd.AddCommand(resultsCmd)
	RootCmd.AddCommand(stdoutCmd)
	RootCmd.AddCommand(stderrCmd)
	RootCmd.AddCommand(artifactsCmd)
	RootCmd.AddCommand(submissionsCmd)
	RootCmd.AddCommand(scoresCmd)
	RootCmd.AddCommand(meCmd)
}
