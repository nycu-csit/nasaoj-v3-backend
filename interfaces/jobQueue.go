package interfaces

import (
	"context"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

// JobQueue is the resource-group mutex a worker consults so at most one job
// per (submission, resource group) runs at a time. Backed by Redis in
// multi-node mode (implements.Redis) or an in-process map in --single-node
// mode (implements.Local).
type JobQueue interface {
	Lock(
		ctx context.Context,
		job *models.Job,
	) error

	Unlock(
		ctx context.Context,
		job *models.Job,
	) error
}
