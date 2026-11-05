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

	// GetTools returns the enabled tools in namespaceIds (nil = every
	// namespace).
	GetTools(
		ctx context.Context,
		namespaceIds []int64,
	) (map[string]models.Tool, error)

	GetToolById(
		ctx context.Context,
		toolId int64,
	) (models.Tool, error)

	// GetAllTools returns every tool revision, enabled or not, in namespaceIds
	// (nil = every namespace).
	GetAllTools(
		ctx context.Context,
		namespaceIds []int64,
	) (map[string]models.Tool, error)

	// UpdateToolState acts on the tool whose sha1 starts with sha1Prefix,
	// looked up only within namespaceIds.
	UpdateToolState(
		ctx context.Context,
		sha1Prefix string,
		namespaceIds []int64,
		isEnabled bool,
	) error

	// DeleteToolAndSubmissions acts on the tool whose sha1 starts with sha1Prefix,
	// looked up only within namespaceIds.
	DeleteToolAndSubmissions(
		ctx context.Context,
		sha1Prefix string,
		namespaceIds []int64,
	) error
}
