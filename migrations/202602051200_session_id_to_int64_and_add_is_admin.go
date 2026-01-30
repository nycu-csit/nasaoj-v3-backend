package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	type oldSession struct {
		bun.BaseModel `bun:"table:session"`
	}

	type newSession struct {
		bun.BaseModel `bun:"table:session"`

		Id int64 `bun:"id,pk"`

		RefreshToken string `bun:"refresh_token"`
		IsValid      bool   `bun:"is_valid"`
	}

	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Print(" [up migration] \n")

		_, err := db.ExecContext(ctx, `
				ALTER TABLE user_resource
				ADD COLUMN is_admin BOOLEAN NOT NULL DEFAULT FALSE
			`)
		if err != nil {
			return fmt.Errorf("failed to add is_admin column: %w", err)
		}

		_, err = db.NewDropTable().
			Model((*oldSession)(nil)).
			IfExists().
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to drop old session table: %w", err)
		}

		_, err = db.NewCreateTable().
			Model((*newSession)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to create new session table: %w", err)
		}

		return nil
	}, nil)
}
