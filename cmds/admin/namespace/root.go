package namespace

import "github.com/spf13/cobra"

var RootCmd = &cobra.Command{
	Use:   "namespace",
	Short: "Manage namespaces",
}

func init() {
	RootCmd.AddCommand(createCmd)
	RootCmd.AddCommand(listCmd)
	RootCmd.AddCommand(enableCmd)
	RootCmd.AddCommand(disableCmd)
	RootCmd.AddCommand(addUsersCmd)
	RootCmd.AddCommand(removeUsersCmd)
	RootCmd.AddCommand(usersCmd)
	RootCmd.AddCommand(importUsersCmd)
}
