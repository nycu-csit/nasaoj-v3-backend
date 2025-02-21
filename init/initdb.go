package init

import (
	"context"

	"github.com/uptrace/bun"
	"gitlab.it.cs.nycu.edu.tw/ytshih/na-judge/models"
)

func InitDB(ctx context.Context, db *bun.DB) error {
	var err error

	err = db.ResetModel(ctx,
		(*models.Job)(nil),
		(*models.Submission)(nil),
	)
	if err != nil {
		panic(err)
	}

	return err
}
