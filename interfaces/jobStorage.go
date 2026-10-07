package interfaces

import (
	"context"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type JobStorage interface {
	// UpdateJob writes job only if its current status is one of from,
	// returning types.JobAlreadySkippedError otherwise.
	UpdateJob(
		ctx context.Context,
		job *models.Job,
		from ...models.JobStatus,
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

	// BlockSubmissionJobs moves a submission's queued jobs (Pending) to Blocked
	// once a worker has claimed the submission.
	BlockSubmissionJobs(
		ctx context.Context,
		subId int64,
	) error

	// ResetSubmissionActiveJobs sets a submission's in-flight jobs
	// (Running/Blocked) back to Pending so a re-claiming worker can re-run them
	// after the submission is requeued.
	ResetSubmissionActiveJobs(
		ctx context.Context,
		subId int64,
	) error
}
