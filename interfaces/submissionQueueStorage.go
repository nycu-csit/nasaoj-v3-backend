package interfaces

import (
	"context"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type SubmissionQueueStorage interface {
	// Enqueue adds an unprocessed submission to the durable queue.
	Enqueue(
		ctx context.Context,
		item models.SubmissionQueueItem,
	) error

	// ClaimNext atomically selects one queued submission whose tag is in
	// `tags`, marks it assigned to `workerName`, and returns it. Returns
	// sql.ErrNoRows when no submission is available. Uses
	// SELECT ... FOR UPDATE SKIP LOCKED so concurrent workers never claim the
	// same submission.
	ClaimNext(
		ctx context.Context,
		workerName string,
		tags []string,
	) (models.SubmissionQueueItem, error)

	// CompleteSubmission removes a finished submission from the queue.
	CompleteSubmission(
		ctx context.Context,
		submissionId int64,
	) error

	// RefreshLease bumps assigned_at for all submissions owned by a worker,
	// keeping the lease alive on heartbeat.
	RefreshLease(
		ctx context.Context,
		workerName string,
	) error

	// ReclaimExpired flips assigned submissions whose lease is older than
	// leaseSeconds back to queued (worker presumed dead) and returns them.
	ReclaimExpired(
		ctx context.Context,
		leaseSeconds int64,
	) ([]models.SubmissionQueueItem, error)

	// SubmissionWorker returns the worker_name currently owning a submission,
	// or an empty string if it is not assigned. Used for the worker-ownership
	// check on forwarded put-args / put-score requests.
	SubmissionWorker(
		ctx context.Context,
		submissionId int64,
	) (string, error)
}
