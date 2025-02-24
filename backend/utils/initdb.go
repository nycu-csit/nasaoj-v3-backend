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
	)
	if err != nil {
		return fmt.Errorf("db reset failed, %s\n", err)
	}

	err = db.NewInsert().Model(&models.Resource{
		Subnet: "192.168.87.0/24",
	}).Scan(ctx)
	if err != nil {
		return fmt.Errorf("db init test value failed%s\n", err)
	}

	return nil
}
