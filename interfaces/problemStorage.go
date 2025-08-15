package interfaces

import (
	"context"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type ProblemStorage interface {
	GetProblem(
		ctx context.Context,
		sha1 string,
	) (models.Problem, error)

	GetTool(
		ctx context.Context,
		sha1 string,
	) (models.Tool, error)

	GetProblemOrTool(
		ctx context.Context,
		sha1 string,
	) (models.Problem, error)

	GetProblemOrToolById(
		ctx context.Context,
		problemId models.ProblemIdType,
	) (models.Problem, error)

	InsertProblem(
		ctx context.Context,
		problem models.Problem,
	) error

	GetProblems(
		ctx context.Context,
	) (map[string]models.Problem, error)

	GetTools(
		ctx context.Context,
	) (map[string]models.Tool, error)

	GetAllProblems(
		ctx context.Context,
	) (map[string]models.Problem, error)

	GetAllTools(
		ctx context.Context,
	) (map[string]models.Tool, error)

	PutProblemState(
		ctx context.Context,
		sha1 string,
		isEnabled bool,
	) error

	InsertContentFiles(
		ctx context.Context,
		contentfiles []models.ContentFile,
	) error

	GetContentFiles(
		ctx context.Context,
		sha1s []string,
	) ([]models.ContentFile, error)

	GetAllContentFiles(
		ctx context.Context,
	) ([]models.ContentFile, error)

	HydrateProblem(
		ctx context.Context,
		problem *models.Problem,
	) error
}
