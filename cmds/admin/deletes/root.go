package deletes

import "github.com/spf13/cobra"

var RootCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete resources",
}

func init() {
	RootCmd.AddCommand(problemCmd)
	RootCmd.AddCommand(toolCmd)
	RootCmd.AddCommand(userCmd)
	RootCmd.AddCommand(submissionCmd)
}
