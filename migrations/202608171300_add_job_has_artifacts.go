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
			ALTER TABLE job
			ADD COLUMN IF NOT EXISTS has_artifacts bool NOT NULL DEFAULT false
		`)
		if err != nil {
			return fmt.Errorf("failed to add has_artifacts column: %w", err)
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		_, err := db.ExecContext(ctx, `
			ALTER TABLE job
			DROP COLUMN IF EXISTS has_artifacts
		`)
		return err
	})
}
