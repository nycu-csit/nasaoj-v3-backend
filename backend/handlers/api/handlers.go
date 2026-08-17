package api

import (
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/implements"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
)

type Handlers struct {
	db          interfaces.Database
	problems    interfaces.ProblemStorage
	tools       interfaces.ToolStorage
	contents    interfaces.ContentFileStorage
	submissions interfaces.SubmissionStorage
	subQueue    interfaces.SubmissionQueueStorage
	jobs        interfaces.JobStorage
	scores      interfaces.ScoreStorage
	namespaces  interfaces.NamespaceStorage
	tokens      interfaces.TokenStorage

	// Worker status/lease bookkeeping: Redis in multi-node mode, an
	// in-process store in --single-node mode.
	workers interfaces.WorkerStatusStorage

	// Object Storage
	s3 interfaces.ObjectStorage
}

func NewHandlers(
	db *implements.Postgres,
	workers interfaces.WorkerStatusStorage,
	s3 *implements.MinIO,
) *Handlers {
	return &Handlers{
		db:          db,
		problems:    db,
		tools:       db,
		contents:    db,
		submissions: db,
		subQueue:    db,
		jobs:        db,
		scores:      db,
		namespaces:  db,
		tokens:      db,

		workers: workers,

		s3: s3,
	}
}
