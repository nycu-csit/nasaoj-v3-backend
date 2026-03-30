package imports

import "github.com/spf13/cobra"

var RootCmd = &cobra.Command{
	Use:   "import",
	Short: "Import resources",
}

func init() {
	RootCmd.AddCommand(ProblemsCmd)
	RootCmd.AddCommand(ToolsCmd)
	RootCmd.AddCommand(usersCmd)
}
