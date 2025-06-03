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
			Table("userresource").
			ColumnExpr("roomid varchar").Exec(ctx)
		if err != nil {
			return err
		}
		_, err = db.NewUpdate().
			Table("userresource").
			Where("TRUE"). // Bun ORM workaround
			Set("roomid = ?", "16").Exec(ctx)
		if err != nil {
			return err
		}
		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Print(" [down migration] ")
		_, err := db.NewDropColumn().
			Table("userresource").
			Column("roomid").Exec(ctx)
		if err != nil {
			return err
		}
		return nil
	})
}
