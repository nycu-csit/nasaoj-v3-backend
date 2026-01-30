package interfaces

import (
	"context"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type ScoreStorage interface {
	GetScores(
		ctx context.Context,
		refId int64,
	) ([]models.UserScore, error)

	GetScoresWithStages(
		ctx context.Context,
		refId int64,
	) ([]models.UserScoreWithStages, error)

	GetScoresWithDeadline(
		ctx context.Context,
		refId int64,
	) ([]models.UserScore, error)

	GetSubmissionScore(
		ctx context.Context,
		subId int64,
	) (models.UserScore, error)
}
