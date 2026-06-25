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

	// KV-Storage only
	jobQue  interfaces.JobQueue
	workers interfaces.WorkerStatusStorage
	tokens  interfaces.TokenStorage

	// Object Storage
	s3 interfaces.ObjectStorage
}

func NewHandlers(
	db *implements.Postgres,
	rdb *implements.Redis,
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

		jobQue:  rdb,
		workers: rdb,
		tokens:  rdb,

		s3: s3,
	}
}
