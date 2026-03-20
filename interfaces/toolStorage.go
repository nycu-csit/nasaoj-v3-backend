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
	) (string, error)

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

	UpdateToolState(
		ctx context.Context,
		sha1Prefix string,
		isEnabled bool,
	) error

	DeleteToolAndSubmissions(
		ctx context.Context,
		sha1Prefix string,
	) error
}
