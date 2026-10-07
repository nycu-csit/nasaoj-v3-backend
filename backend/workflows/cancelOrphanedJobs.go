package workflows

import (
	"context"
	"errors"
	"fmt"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"go.uber.org/zap"
)

// CancelOrphanedJobs marks a submission's Running jobs that no live worker
// tracks as Cancelled. A cancelled submission's Running jobs are normally
// finished by the owning worker (signalled via the heartbeat), but a job can
// be left Running with no worker behind it - e.g. marshal committed the status
// change and then failed - and would otherwise never become terminal.
func CancelOrphanedJobs(
	ctx context.Context,
	submission models.Submission,
	workers interfaces.WorkerStatusStorage,
	jobs interfaces.JobStorage,
) error {
	var running []*models.Job
	for _, job := range submission.Jobs {
		if job.Status == models.StatusRunning {
			running = append(running, job)
		}
	}
	if len(running) == 0 {
		return nil
	}

	workerMap, err := workers.GetWorkers(ctx)
	if err != nil {
		return fmt.Errorf("failed to get workers: %w", err)
	}
	tracked := make(map[string]struct{})
	for name := range workerMap {
		jobIds, err := workers.GetWorkerJobs(ctx, name)
		if err != nil {
			return fmt.Errorf("failed to get jobs of worker %s: %w", name, err)
		}
		for _, jobId := range jobIds {
			tracked[jobId] = struct{}{}
		}
	}

	for _, job := range running {
		if _, ok := tracked[job.Id()]; ok {
			continue
		}
		tracing.Logger.Ctx(ctx).
			Warn("cancelling running job no worker tracks",
				zap.String("job.id", job.Id()))
		job.Status = models.StatusCancelled
		err := jobs.UpdateJob(ctx, job, models.StatusRunning)
		if err != nil && !errors.Is(err, types.JobAlreadySkippedError) {
			return fmt.Errorf("failed to cancel job %s: %w", job.Id(), err)
		}
	}

	return nil
}
