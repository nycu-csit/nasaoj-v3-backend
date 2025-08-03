package imports

import "github.com/spf13/cobra"

var usersCmd = &cobra.Command{
	Use:   "users",
	Short: "Import users",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}
