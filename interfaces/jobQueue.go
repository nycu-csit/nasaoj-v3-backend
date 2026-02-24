package interfaces

import (
	"context"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type JobQueue interface {
	GetJob(
		ctx context.Context,
		tag string,
	) (models.Job, error)

	PutJob(
		ctx context.Context,
		job models.Job,
		tag string,
	) error

	SetJobCnt(
		ctx context.Context,
		stageId string,
		cnt int,
	) error

	SetDepCnt(
		ctx context.Context,
		stageId string,
		cnt int,
	) error

	JobDone(
		ctx context.Context,
		stageId string,
	) (int64, error)

	StageDone(
		ctx context.Context,
		stageId string,
	) (int64, error)

	Lock(
		ctx context.Context,
		job *models.Job,
	) error

	Unlock(
		ctx context.Context,
		job *models.Job,
	) error

	PutRetryJob(
		ctx context.Context,
		job models.Job,
		tag string,
	) error

	GetRetryJob(
		ctx context.Context,
		tag string,
	) (models.Job, error)
}
