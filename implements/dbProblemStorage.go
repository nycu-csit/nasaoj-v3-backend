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

func (self *DBProblemStorage) GetTool(
	ctx context.Context,
	sha1 string,
) (models.Tool, error) {
	return self.db.GetTool(ctx, sha1)
}

func (self *DBProblemStorage) InsertProblem(
	ctx context.Context,
	problem models.Problem,
) error {
	return self.db.InsertProblem(ctx, problem)
}

func (self *DBProblemStorage) InsertContentFiles(
	ctx context.Context,
	contentFiles []models.ContentFile,
) error {
	return self.db.InsertContentFiles(ctx, contentFiles)
}

func (self *DBProblemStorage) GetContentFiles(
	ctx context.Context,
	sha1s []string,
) ([]models.ContentFile, error) {
	return self.db.GetContentFiles(ctx, sha1s)
}

func (self *DBProblemStorage) GetAllContentFiles(
	ctx context.Context,
) ([]models.ContentFile, error) {
	return self.db.GetAllContentFiles(ctx)
}

func (self *DBProblemStorage) GetProblems(
	ctx context.Context,
) (map[string]models.Problem, error) {
	return self.db.GetProblems(ctx)
}

func (self *DBProblemStorage) GetTools(
	ctx context.Context,
) (map[string]models.Tool, error) {
	return self.db.GetTools(ctx)
}

func (self *DBProblemStorage) GetProblemOrTool(
	ctx context.Context,
	sha1 string,
) (models.Problem, error) {
	return self.db.GetProblemOrTool(ctx, sha1)
}

func (self *DBProblemStorage) GetProblemOrToolById(
	ctx context.Context,
	problemId models.ProblemIdType,
) (models.Problem, error) {
	return self.db.GetProblemOrToolById(ctx, problemId)
}

func (self *DBProblemStorage) GetAllProblems(
	ctx context.Context,
) (map[string]models.Problem, error) {
	return self.db.GetAllProblems(ctx)
}

func (self *DBProblemStorage) GetAllTools(
	ctx context.Context,
) (map[string]models.Tool, error) {
	return self.db.GetAllTools(ctx)
}

func (self *DBProblemStorage) PutProblemState(
	ctx context.Context,
	sha1 string,
	isEnabled bool,
) error {
	return self.db.UpdateProblem(ctx, sha1, isEnabled)
}

func (self *DBProblemStorage) HydrateProblem(
	ctx context.Context,
	problem *models.Problem,
) error {
	return self.db.HydrateProblem(ctx, problem)
}
