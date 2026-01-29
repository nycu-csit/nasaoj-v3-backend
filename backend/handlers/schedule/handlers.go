package schedule

import (
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/implements"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
)

type Handlers struct {
	db          interfaces.Database
	jobs        interfaces.JobStorage
	problems    interfaces.ProblemStorage
	contents    interfaces.ContentFileStorage
	submissions interfaces.SubmissionStorage

	jobQue  interfaces.JobQueue
	tokens  interfaces.TokenStorage
	workers interfaces.WorkerStatusStorage

	s3 interfaces.ObjectStorage
}

func NewHandlers(
	db *implements.Postgres,
	kv *implements.Redis,
	s3 *implements.MinIO,
) *Handlers {
	return &Handlers{
		db:          db,
		jobs:        db,
		problems:    db,
		contents:    db,
		submissions: db,

		jobQue:  kv,
		tokens:  kv,
		workers: kv,

		s3: s3,
	}
}
