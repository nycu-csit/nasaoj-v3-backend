package schedule

import (
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/implements"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
)

type Handlers struct {
	db          interfaces.Database
	jobs        interfaces.JobStorage
	problems    interfaces.ProblemStorage
	tools       interfaces.ToolStorage
	contents    interfaces.ContentFileStorage
	submissions interfaces.SubmissionStorage
	subQueue    interfaces.SubmissionQueueStorage
	namespaces  interfaces.NamespaceStorage

	// Resource-group locks and worker status/lease bookkeeping: Redis in
	// multi-node mode, in-process stores in --single-node mode.
	jobQue  interfaces.JobQueue
	workers interfaces.WorkerStatusStorage

	s3 interfaces.ObjectStorage
}

func NewHandlers(
	db *implements.Postgres,
	jobQue interfaces.JobQueue,
	workers interfaces.WorkerStatusStorage,
	s3 *implements.MinIO,
) *Handlers {
	return &Handlers{
		db:          db,
		jobs:        db,
		problems:    db,
		tools:       db,
		contents:    db,
		submissions: db,
		subQueue:    db,
		namespaces:  db,

		jobQue:  jobQue,
		workers: workers,

		s3: s3,
	}
}
