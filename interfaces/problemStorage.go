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
		namespaceId int64,
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
	) (string, error)

	// UpdateProblemState acts on the problem whose sha1 starts with sha1Prefix,
	// looked up only within namespaceIds.
	UpdateProblemState(
		ctx context.Context,
		sha1Prefix string,
		namespaceIds []int64,
		isEnabled bool,
	) error

	// DeleteProblemAndSubmissions acts on the problem whose sha1 starts with sha1Prefix,
	// looked up only within namespaceIds.
	DeleteProblemAndSubmissions(
		ctx context.Context,
		sha1Prefix string,
		namespaceIds []int64,
	) error
}
