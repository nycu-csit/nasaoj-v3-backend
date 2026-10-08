package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Print(" [up migration] \n")

		// Default namespace membership used to be implicit: a user with no
		// user_namespace rows was treated as a member of default. It is now
		// an explicit row, so give one to exactly those users - their access
		// is unchanged. Global admins keep mirroring their flag, as in the
		// original namespace backfill.
		_, err := db.ExecContext(ctx, `
			INSERT INTO user_namespace (user_id, namespace_id, is_admin)
			SELECT u.id, 1, u.is_admin FROM user_resource u
			WHERE NOT EXISTS (
				SELECT 1 FROM user_namespace m WHERE m.user_id = u.id
			)
			ON CONFLICT (user_id, namespace_id) DO NOTHING
		`)
		if err != nil {
			return fmt.Errorf("failed to backfill default namespace membership: %w", err)
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		// Nothing to undo: with the implicit fallback back in place, these
		// rows grant exactly what the fallback would.
		return nil
	})
}
