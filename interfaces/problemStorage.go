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

	InsertProblem(
		ctx context.Context,
		problem models.Problem,
	) error

	GetAllProblems(
		ctx context.Context,
	) (map[string]models.Problem, error)

	PutProblemState(
		ctx context.Context,
		sha1 string,
		isEnabled bool,
	) error
}
