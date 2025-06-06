package interfaces

import (
	"context"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type ScoreCache interface {
	Put(
		ctx context.Context,
		problem models.Problem,
		stageName string,
		csId string,
		score int64,
	) error

	Get(
		ctx context.Context,
		problem models.Problem,
		isAdmin bool,
	) ([]models.UserScore, error)

	Reload(
		ctx context.Context,
		db Database,
		problems models.ProblemRepository,
	) error
}
