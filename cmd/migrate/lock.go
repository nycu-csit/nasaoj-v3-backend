package migrate

import (
	"github.com/spf13/cobra"
)

var lockCmd = &cobra.Command{
	Use:   "lock",
	Short: "Lock migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		return migrator.Lock(cmd.Context())
	},
}
