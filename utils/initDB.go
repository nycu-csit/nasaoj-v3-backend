package utils

import (
	"context"

	"github.com/uptrace/bun"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

func InitDB(ctx context.Context, db *bun.DB) error {
	return db.ResetModel(ctx,
		(*models.ProblemMarshaled)(nil),
		(*models.ToolMarshaled)(nil),
		(*models.ContentFile)(nil),
		(*models.Job)(nil),
		(*models.Submission)(nil),
		(*models.UserResource)(nil),
		(*models.PlainCred)(nil),
		(*models.Session)(nil),
	)
}
