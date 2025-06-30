package implements

import (
	"context"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type DBProblemStorage struct {
	db interfaces.Database
	// TODO: add cache
}

func NewDBProblemStorage(db interfaces.Database) *DBProblemStorage {
	return &DBProblemStorage{db: db}
}

func (self *DBProblemStorage) GetProblem(
	ctx context.Context,
	sha1 string,
) (models.Problem, error) {
	return self.db.GetProblem(ctx, sha1)
}

func (self *DBProblemStorage) InsertProblem(
	ctx context.Context,
	problem models.Problem,
) error {
	return self.db.InsertProblem(ctx, problem)
}
