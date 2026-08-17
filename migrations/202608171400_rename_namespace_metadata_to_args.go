package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Print(" [up migration] \n")

		_, err := db.ExecContext(ctx, `
			ALTER TABLE namespace
			RENAME COLUMN metadata TO args
		`)
		if err != nil {
			return fmt.Errorf("failed to rename namespace.metadata to args: %w", err)
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		_, err := db.ExecContext(ctx, `
			ALTER TABLE namespace
			RENAME COLUMN args TO metadata
		`)
		return err
	})
}
