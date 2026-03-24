package interfaces

import (
	"context"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type JobStorage interface {
	UpdateJob(
		ctx context.Context,
		job *models.Job,
	) error

	SkipJobs(
		ctx context.Context,
		subId int64,
	) error

	SkipJobWithStages(
		ctx context.Context,
		subId int64,
		stages []string,
	) error

	TimeoutJobById(
		ctx context.Context,
		jobId string,
	) error

	GetRunningJobs(
		ctx context.Context,
	) ([]models.Job, error)
}
