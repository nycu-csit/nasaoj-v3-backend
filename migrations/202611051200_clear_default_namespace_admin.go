package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Print(" [up migration] \n")

		// The default namespace membership used to mirror the global admin
		// flag (namespace backfill, explicit default membership backfill,
		// and user creation), but demoting a global admin only cleared
		// user_resource.is_admin - leaving ex-admins as namespace admins of
		// default. Global admins bypass namespace checks anyway, so drop the
		// namespace admin role from default for everyone.
		_, err := db.ExecContext(ctx, `
			UPDATE user_namespace SET is_admin = false
			WHERE namespace_id = 1 AND is_admin
		`)
		if err != nil {
			return fmt.Errorf("failed to clear default namespace admins: %w", err)
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		// Not reversible: which rows were mirrored and which were granted
		// on purpose is not recorded.
		return nil
	})
}
