package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

func init() {
	type oldUserResource struct {
		bun.BaseModel `bun:"table:userresource"`

		CsId string            `bun:"cs_id,pk"`
		Args map[string]string `bun:"args"`
	}

	type newUserResource struct {
		bun.BaseModel `bun:"table:user_resource"`

		Id       int64             `bun:"id,pk,autoincrement"`
		Username string            `bun:"username,unique"`
		Args     map[string]string `bun:"args"`

		Source   int    `bun:"source"`
		SourceId string `bun:"source_id"`
	}

	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Print(" [up migration] \n")

		var oldUsers []oldUserResource
		err := db.NewSelect().
			Model(&oldUsers).
			Scan(ctx)
		if err != nil {
			return err
		}

		err = db.ResetModel(ctx,
			(*newUserResource)(nil))
		if err != nil {
			return err
		}

		for _, user := range oldUsers {
			_, err := db.NewInsert().
				Model(&newUserResource{
					Username: user.CsId,
					Args:     user.Args,

					Source:   models.LoginSourceCSSSO,
					SourceId: user.CsId,
				}).
				Exec(ctx)
			if err != nil {
				return err
			}
		}

		_, err = db.NewDropTable().
			Model((*oldUserResource)(nil)).
			Exec(ctx)
		if err != nil {
			return err
		}

		return nil
	}, nil)
}
