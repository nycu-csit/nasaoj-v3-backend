package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	type privileges struct {
		OwnerId          int64 `json:"ownerId"`
		CanAPIRead       int   `json:"canAPIRead"`
		CanAPIWrite      int   `json:"canAPIWrite"`
		CanUserArgsWrite int   `json:"canUserArgsWrite"`
		CanScheduleAPI   int   `json:"canScheduleAPI"`
	}

	// token is the durable store for static bearer tokens, replacing the
	// Redis hash they used to live in. SHA256 (the hex digest of the raw
	// token) is the primary key and lookup key; the raw token is never
	// stored.
	type token struct {
		bun.BaseModel `bun:"table:token"`

		SHA256      string     `bun:"sha256,pk"`
		UserId      int64      `bun:"user_id"`
		Description string     `bun:"description"`
		Privs       privileges `bun:"privs"`
		IsEnabled   bool       `bun:"is_enabled"`
		CreatedAt   int64      `bun:"created_at"`
	}

	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Print(" [up migration] \n")

		if _, err := db.NewCreateTable().
			Model((*token)(nil)).
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to create token table: %w", err)
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		_, err := db.NewDropTable().
			Model((*token)(nil)).IfExists().Exec(ctx)
		return err
	})
}
