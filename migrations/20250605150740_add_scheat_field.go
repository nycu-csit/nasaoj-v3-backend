package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Print(" [up migration] ")
		_, err := db.NewAddColumn().
			Table("job").
			ColumnExpr("scheat bigint").Exec(ctx)
		return err
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Print(" [down migration] ")
		_, err := db.NewDropColumn().
			Table("job").
			Column("scheat").Exec(ctx)
		return err
	})
}
