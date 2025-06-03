package migrate

import "github.com/spf13/cobra"

var unlockCmd = &cobra.Command{
	Use:   "unlock",
	Short: "Unlock migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		return migrator.Unlock(cmd.Context())
	},
}
