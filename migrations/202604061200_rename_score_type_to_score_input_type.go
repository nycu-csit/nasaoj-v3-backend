package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Print(" [up migration] \n")

		// Update the JSON payload in the problem table to rename scoreType to scoreInputType
		_, err := db.ExecContext(ctx, `
			UPDATE problem
			SET payload = REPLACE(payload, '"scoreType":', '"scoreInputType":')
			WHERE payload LIKE '%"scoreType":%'
		`)
		if err != nil {
			return fmt.Errorf("failed to rename scoreType to scoreInputType in problem payload: %w", err)
		}

		return nil
	}, nil)
}
