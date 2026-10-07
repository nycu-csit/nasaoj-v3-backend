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

// CancelOrphanedJobs marks a submission's Running jobs that its owning worker
// doesn't track as Cancelled. A cancelled submission's Running jobs are
// normally finished by the owning worker (signalled via the heartbeat), but a
// job can be left Running with nobody behind it - e.g. its worker tracking was
// lost - and would otherwise never become terminal.
//
// Marshal tracks a job on its worker before making it Running, so a job seen
// Running here is always visible in the owner's tracked set unless it really
// is orphaned.
func CancelOrphanedJobs(
	ctx context.Context,
	submission models.Submission,
	subQueue interfaces.SubmissionQueueStorage,
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

	// Only the worker holding the submission can be running its jobs. Ask
	// it directly rather than scanning registered workers: an owner that
	// hasn't heartbeated yet is not registered but does track its jobs.
	owner, err := subQueue.SubmissionWorker(ctx, submission.Id)
	if err != nil {
		return fmt.Errorf("failed to get submission owner: %w", err)
	}
	tracked := make(map[string]struct{})
	if owner != "" {
		jobIds, err := workers.GetWorkerJobs(ctx, owner)
		if err != nil {
			return fmt.Errorf("failed to get jobs of worker %s: %w", owner, err)
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
			Warn("cancelling running job its owner doesn't track",
				zap.String("job.id", job.Id()),
				zap.String("owner", owner))
		job.Status = models.StatusCancelled
		err := jobs.UpdateJob(ctx, job, models.StatusRunning)
		if err != nil && !errors.Is(err, types.JobAlreadySkippedError) {
			return fmt.Errorf("failed to cancel job %s: %w", job.Id(), err)
		}
	}

	return nil
}
