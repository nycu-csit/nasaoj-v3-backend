package implements

import (
	"context"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type DBToolStorage struct {
	db interfaces.Database
	// TODO: add cache
}

func NewDBToolStorage(db interfaces.Database) *DBToolStorage {
	return &DBToolStorage{db: db}
}

func (self *DBToolStorage) GetTool(
	ctx context.Context,
	sha1 string,
) (models.Tool, error) {
	return self.db.GetTool(ctx, sha1)
}

func (self *DBToolStorage) GetToolById(
	ctx context.Context,
	toolId int64,
) (models.Tool, error) {
	return self.db.GetToolById(ctx, toolId)
}

func (self *DBToolStorage) InsertTool(
	ctx context.Context,
	tool models.Tool,
) error {
	return self.db.InsertTool(ctx, tool)
}

func (self *DBToolStorage) InsertContentFiles(
	ctx context.Context,
	contentFiles []models.ContentFile,
) error {
	return self.db.InsertContentFiles(ctx, contentFiles)
}

func (self *DBToolStorage) GetContentFiles(
	ctx context.Context,
	sha1s []string,
) ([]models.ContentFile, error) {
	return self.db.GetContentFiles(ctx, sha1s)
}

func (self *DBToolStorage) GetTools(
	ctx context.Context,
) (map[string]models.Tool, error) {
	return self.db.GetTools(ctx)
}

func (self *DBToolStorage) GetAllTools(
	ctx context.Context,
) (map[string]models.Tool, error) {
	return self.db.GetAllTools(ctx)
}

func (self *DBToolStorage) PutToolState(
	ctx context.Context,
	sha1 string,
	isEnabled bool,
) error {
	return self.db.UpdateTool(ctx, sha1, isEnabled)
}

func (self *DBToolStorage) HydrateTool(
	ctx context.Context,
	tool *models.Tool,
) error {
	return self.db.HydrateTool(ctx, tool)
}
