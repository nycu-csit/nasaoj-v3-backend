package crons

import (
	"context"
	"time"

	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"go.uber.org/zap"
)

func CleanOrphanedJobs(
	ctx context.Context,
	workers interfaces.WorkerStatusStorage,
	jobs interfaces.JobStorage,
) {
	ticker := time.NewTicker(
		time.Duration(viper.GetInt64("worker-timeout")) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cleanOrphanedJobs(ctx, workers, jobs)
		}
	}
}

func cleanOrphanedJobs(
	ctx context.Context,
	workers interfaces.WorkerStatusStorage,
	jobs interfaces.JobStorage,
) {
	tracing.Logger.Ctx(ctx).
		Info("run cleanOrphanedJobs")

	// Get all running jobs from database
	runningJobs, err := jobs.GetRunningJobs(ctx)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to get running jobs",
				zap.Error(err))
		return
	}

	if len(runningJobs) == 0 {
		return
	}

	// Get all workers and their tracked jobs
	workerMap, err := workers.GetWorkers(ctx)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to get workers",
				zap.Error(err))
		return
	}

	// Build a set of all job IDs tracked by workers
	trackedJobs := make(map[string]struct{})
	for name := range workerMap {
		jobIds, err := workers.GetWorkerJobs(ctx, name)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to get worker jobs",
					zap.String("worker", name),
					zap.Error(err))
			continue
		}
		for _, jobId := range jobIds {
			trackedJobs[jobId] = struct{}{}
		}
	}

	// Find orphaned jobs (running but not tracked by any worker)
	for _, job := range runningJobs {
		jobId := job.Id()
		if _, tracked := trackedJobs[jobId]; !tracked {
			tracing.Logger.Ctx(ctx).
				Warn("found orphaned running job, marking as missing",
					zap.String("job.id", jobId))

			if err := jobs.TimeoutJobById(ctx, jobId); err != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to timeout orphaned job",
						zap.String("job.id", jobId),
						zap.Error(err))
			}
		}
	}
}
