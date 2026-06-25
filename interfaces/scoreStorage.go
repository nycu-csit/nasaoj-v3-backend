package interfaces

import (
	"context"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type ScoreStorage interface {
	GetScores(
		ctx context.Context,
		namespaceId int64,
		refId int64,
	) ([]models.UserScore, error)

	GetScoresWithStages(
		ctx context.Context,
		namespaceId int64,
		refId int64,
	) ([]models.UserScoreWithStages, error)

	GetScoresWithDeadline(
		ctx context.Context,
		namespaceId int64,
		refId int64,
	) ([]models.UserScore, error)

	GetSubmissionScore(
		ctx context.Context,
		subId int64,
	) (models.UserScore, error)

	// GetNamespaceScoreboard returns each member's total score across all
	// enabled problems in the namespace.
	GetNamespaceScoreboard(
		ctx context.Context,
		namespaceId int64,
	) ([]models.ScoreboardEntry, error)
}
