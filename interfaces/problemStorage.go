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

	GetProblems(
		ctx context.Context,
	) (map[string]models.Problem, error)

	GetProblemById(
		ctx context.Context,
		problemId int64,
	) (models.Problem, error)

	GetAllProblems(
		ctx context.Context,
	) (map[string]models.Problem, error)

	GetProblemScores(
		ctx context.Context,
		sha1s []string,
	) (map[string]int64, error)

	InsertProblem(
		ctx context.Context,
		problem models.Problem,
	) error

	UpdateProblemState(
		ctx context.Context,
		sha1 string,
		isEnabled bool,
	) error

	DeleteProblemAndSubmissions(
		ctx context.Context,
		sha1 string,
	) error
}
