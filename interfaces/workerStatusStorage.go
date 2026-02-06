package interfaces

import (
	"context"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type WorkerStatusStorage interface {
	GetWorkers(
		ctx context.Context,
	) (models.WorkerStatusMap, error)

	UpdateWorkerStatus(
		ctx context.Context,
		name string,
		ip string,
		tags []string,
	) error

	UpdateLastAccess(
		ctx context.Context,
		name string,
	) error

	DeleteWorker(
		ctx context.Context,
		name string,
	) error
}
