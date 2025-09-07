package interfaces

import (
	"context"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type ContentFileStorage interface {
	GetContentFiles(
		ctx context.Context,
		sha1s []string,
	) ([]models.ContentFile, error)

	GetAllContentFiles(
		ctx context.Context,
	) ([]models.ContentFile, error)

	InsertContentFiles(
		ctx context.Context,
		contentFiles []models.ContentFile,
	) error

	HydrateContentFiles(
		ctx context.Context,
		contentFiles *[]models.ContentFile,
	) error

	HydrateProblem(
		ctx context.Context,
		problem *models.Problem,
	) error

	HydrateTool(
		ctx context.Context,
		tool *models.Tool,
	) error
}
