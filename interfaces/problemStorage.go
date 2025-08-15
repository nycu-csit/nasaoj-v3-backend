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
}
