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

// ReclaimSubmissions periodically flips submissions leased to dead workers back
// to the queue so another worker can pick them up. Replaces the per-job orphan
// cleanup now that scheduling is submission-level.
func ReclaimSubmissions(
	ctx context.Context,
	subQueue interfaces.SubmissionQueueStorage,
	jobs interfaces.JobStorage,
) {
	interval := viper.GetInt64("worker-timeout")
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			reclaimSubmissions(ctx, subQueue, jobs, interval)
		}
	}
}

func reclaimSubmissions(
	pctx context.Context,
	subQueue interfaces.SubmissionQueueStorage,
	jobs interfaces.JobStorage,
	leaseSeconds int64,
) {
	ctx, span := tracing.Tracer.Start(pctx, "cron.ReclaimSubmissions")
	defer span.End()
	span.SetAttributes(
		attribute.String("faas.trigger", "timer"),
		attribute.String("faas.name", "nasaoj-backend"),
	)

	// Lease window is generous: 3x the heartbeat-driven worker timeout.
	reclaimed, err := subQueue.ReclaimExpired(ctx, leaseSeconds*3)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to reclaim expired submissions",
				zap.Error(err))
		return
	}

	for _, item := range reclaimed {
		tracing.Logger.Ctx(ctx).
			Warn("reclaimed submission from dead worker",
				zap.Int64("submission.id", item.SubmissionId),
				zap.String("worker.name", item.WorkerName))

		if err := jobs.ResetSubmissionActiveJobs(ctx, item.SubmissionId); err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to reset jobs for reclaimed submission",
					zap.Int64("submission.id", item.SubmissionId),
					zap.Error(err))
		}
	}
}
