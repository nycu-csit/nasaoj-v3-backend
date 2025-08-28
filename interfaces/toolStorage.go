package interfaces

import (
	"context"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type ToolStorage interface {
	GetTool(
		ctx context.Context,
		sha1 string,
	) (models.Tool, error)

	InsertTool(
		ctx context.Context,
		tool models.Tool,
	) error

	GetTools(
		ctx context.Context,
	) (map[string]models.Tool, error)

	GetToolById(
		ctx context.Context,
		toolId int64,
	) (models.Tool, error)

	GetAllTools(
		ctx context.Context,
	) (map[string]models.Tool, error)

	PutToolState(
		ctx context.Context,
		sha1 string,
		isEnabled bool,
	) error

	InsertContentFiles(
		ctx context.Context,
		contentFiles []models.ContentFile,
	) error

	GetContentFiles(
		ctx context.Context,
		sha1s []string,
	) ([]models.ContentFile, error)

	HydrateTool(
		ctx context.Context,
		tool *models.Tool,
	) error
}
