package imports

import (
	"github.com/spf13/cobra"
)

var usersCmd = &cobra.Command{
	Use:   "users",
	Short: "Import users",
}

func init() {
	usersCmd.AddCommand(usersPlainCmd)
	usersCmd.AddCommand(usersCSSSOCmd)
}
