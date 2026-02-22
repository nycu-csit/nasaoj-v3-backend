package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	type plainCred struct {
		bun.BaseModel `bun:"table:plain_cred"`

		ResourceId int64  `bun:"resource_id,pk,autoincrement"`
		Passhash   string `bun:"passhash"`
	}
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Print(" [up migration] \n")
		return db.ResetModel(ctx,
			(*plainCred)(nil))
	}, nil)
}
