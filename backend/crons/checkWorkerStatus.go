package crons

import (
	"context"
	"time"

	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

func CheckWorkerStatus(
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
			checkWorkers(ctx, workers, jobs)
		}
	}
}

func checkWorkers(
	pctx context.Context,
	workers interfaces.WorkerStatusStorage,
	jobs interfaces.JobStorage,
) {
	ctx, span := tracing.Tracer.Start(pctx, "cron.CheckWorkerStatus")
	defer span.End()
	span.SetAttributes(
		attribute.String("faas.trigger", "timer"),
		attribute.String("faas.name", "nasaoj-backend"),
	)

	tracing.Logger.Ctx(ctx).
		Debug("run checkWorkers")

	workerMap, err := workers.GetWorkers(ctx)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to get workers",
				zap.Error(err))
		return
	}

	timeout := viper.GetInt64("worker-timeout")
	now := time.Now().Unix()

	for name, status := range workerMap {
		if now-status.LastAccess > timeout {
			tracing.Logger.Ctx(ctx).
				Warn("worker timeout, removing worker and timing out jobs",
					zap.String("worker", name),
					zap.Int64("lastAccess", status.LastAccess))

			// Get the jobs this worker was handling
			jobIds, err := workers.GetWorkerJobs(ctx, name)
			if err != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to get worker jobs",
						zap.String("worker", name),
						zap.Error(err))
				continue
			}

			// Timeout all jobs
			for _, jobId := range jobIds {
				if err := jobs.TimeoutJobById(ctx, jobId); err != nil {
					tracing.Logger.Ctx(ctx).
						Error("failed to timeout job",
							zap.String("worker", name),
							zap.String("job.id", jobId),
							zap.Error(err))
				}
			}

			// Delete the worker
			if err := workers.DeleteWorker(ctx, name); err != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to delete worker",
						zap.String("worker", name),
						zap.Error(err))
			}
		}
	}
}
