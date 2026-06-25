package models

import "github.com/uptrace/bun"

type SubQueueStatus = int

const (
	SubQueued   SubQueueStatus = 0
	SubAssigned SubQueueStatus = 1
)

// SubmissionQueueItem is the durable, claimable queue of unprocessed
// submissions. The backend schedules one submission (problem/tool) at a time:
// a worker claims a row, owns the whole submission, and resolves the per-stage
// dependencies itself. `WorkerName` is the "which submission is in which worker"
// mapping.
type SubmissionQueueItem struct {
	bun.BaseModel `bun:"table:submission_queue"`

	SubmissionId int64   `json:"submissionId" bun:"submission_id,pk"`
	UserId       int64   `json:"userId" bun:"user_id"`
	ProblemId    int64   `json:"problemId" bun:"problem_id"`
	ProblemSHA1  string  `json:"problemSHA1" bun:"problem_sha1"`
	Type         RefType `json:"type" bun:"type"`
	Tag          string  `json:"tag" bun:"tag"`

	Status     SubQueueStatus `json:"status" bun:"status"`
	WorkerName string         `json:"workerName" bun:"worker_name,nullzero"`
	AssignedAt int64          `json:"assignedAt" bun:"assigned_at,nullzero"`
	EnqueuedAt int64          `json:"enqueuedAt" bun:"enqueued_at"`
}
