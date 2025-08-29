package implements

import (
	"context"

	"github.com/uptrace/bun"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type SqliteDatabase struct {
	*BunDatabase
}

func NewSqliteDatabase(db *bun.DB) *SqliteDatabase {
	return &SqliteDatabase{
		BunDatabase: NewBunDatabase(db),
	}
}

func (self *SqliteDatabase) GetSubmission(
	ctx context.Context,
	subId int64,
) (models.Submission, error) {
	ret := models.Submission{Id: subId}
	if err := self.submissionQueryHook(ctx, subId); err != nil {
		return models.Submission{}, err
	}
	err := self.db.NewSelect().
		Model(&ret).
		WherePK().
		Relation("Jobs").
		Scan(ctx)
	return ret, err
}
