package utils

import (
	"context"

	"github.com/uptrace/bun"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

func InitDB(ctx context.Context, db *bun.DB) error {
	var err error

	err = db.ResetModel(ctx,
		(*models.ProblemMarshaled)(nil),
		(*models.ContentFile)(nil),
		(*models.Job)(nil),
		(*models.Submission)(nil),
		(*models.UserResource)(nil),
		(*models.Session)(nil),
	)
	if err != nil {
		panic(err)
	}

	return nil
}
