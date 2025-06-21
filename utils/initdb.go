package utils

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

func InitDB(ctx context.Context, db *bun.DB) error {
	var err error

	err = db.ResetModel(ctx,
		(*models.Job)(nil),
		(*models.Submission)(nil),
		(*models.Resource)(nil),
		(*models.UserResource)(nil),
		(*models.Session)(nil),
	)
	if err != nil {
		return fmt.Errorf("db reset failed, %s\n", err)
	}

	return nil
}
