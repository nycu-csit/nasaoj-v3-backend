package migrates

import (
	"fmt"

	"github.com/spf13/cobra"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "migrate up",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		if err := migrator.Lock(ctx); err != nil {
			return err
		}
		defer migrator.Unlock(ctx)

		group, err := migrator.Migrate(ctx)
		if err != nil {
			return err
		}

		if group.IsZero() {
			fmt.Printf("there are no new migrations to run\n")
			return nil
		}

		fmt.Printf("migrated to %s\n", group)
		return nil
	},
}
