package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	type namespace struct {
		bun.BaseModel `bun:"table:namespace"`

		Id        int64             `bun:"id,pk,autoincrement"`
		Name      string            `bun:"name,unique"`
		Title     string            `bun:"title"`
		Metadata  map[string]string `bun:"metadata"`
		CreatedAt int64             `bun:"created_at"`
		IsEnabled bool              `bun:"is_enabled"`
	}

	type userNamespace struct {
		bun.BaseModel `bun:"table:user_namespace"`

		UserId      int64 `bun:"user_id,pk"`
		NamespaceId int64 `bun:"namespace_id,pk"`
		IsAdmin     bool  `bun:"is_admin"`
	}

	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Print(" [up migration] \n")

		if _, err := db.NewCreateTable().
			Model((*namespace)(nil)).
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to create namespace table: %w", err)
		}

		if _, err := db.NewCreateTable().
			Model((*userNamespace)(nil)).
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to create user_namespace table: %w", err)
		}

		// Seed the default namespace as id 1 (fresh table -> first insert is 1).
		_, err := db.NewInsert().
			Model(&namespace{
				Name:      "default",
				Title:     "Default",
				IsEnabled: true,
			}).
			On("CONFLICT (name) DO NOTHING").
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to seed default namespace: %w", err)
		}

		// Add namespace_id to all namespaced resources, defaulting existing rows
		// to the default namespace.
		for _, table := range []string{
			"problem", "tool", "submission", "job", "submission_queue",
		} {
			if _, err := db.ExecContext(ctx, fmt.Sprintf(
				`ALTER TABLE %q ADD COLUMN IF NOT EXISTS namespace_id bigint NOT NULL DEFAULT 1`,
				table)); err != nil {
				return fmt.Errorf("failed to add namespace_id to %s: %w", table, err)
			}
		}

		// Backfill membership: every existing user joins the default namespace,
		// carrying their global admin flag as the per-namespace admin role.
		if _, err := db.ExecContext(ctx, `
			INSERT INTO user_namespace (user_id, namespace_id, is_admin)
			SELECT id, 1, is_admin FROM user_resource
			ON CONFLICT (user_id, namespace_id) DO NOTHING
		`); err != nil {
			return fmt.Errorf("failed to backfill user_namespace: %w", err)
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		for _, table := range []string{
			"problem", "tool", "submission", "job", "submission_queue",
		} {
			if _, err := db.ExecContext(ctx, fmt.Sprintf(
				`ALTER TABLE %q DROP COLUMN IF EXISTS namespace_id`,
				table)); err != nil {
				return err
			}
		}
		if _, err := db.NewDropTable().
			Model((*userNamespace)(nil)).IfExists().Exec(ctx); err != nil {
			return err
		}
		_, err := db.NewDropTable().
			Model((*namespace)(nil)).IfExists().Exec(ctx)
		return err
	})
}
