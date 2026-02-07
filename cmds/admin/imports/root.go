package imports

import "github.com/spf13/cobra"

var RootCmd = &cobra.Command{
	Use:   "import",
	Short: "Import resources",
}

func init() {
	RootCmd.AddCommand(problemsCmd)
	RootCmd.AddCommand(toolsCmd)
	RootCmd.AddCommand(usersCmd)
}
