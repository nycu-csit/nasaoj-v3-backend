package workflows

import (
	"context"
	"fmt"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/worker/cri"
	"go.opentelemetry.io/otel/attribute"
)

func RunJob(
	pctx context.Context,
	retCh chan error,
	job *models.JobWithFile,
	cancelRegistry *CancelRegistry,
	criClient *cri.Client,
	workerBinary string,
) {
	ctx, span := tracing.Tracer.Start(pctx, fmt.Sprintf("worker.RunJob/%s", job.Id()))
	defer span.End()

	span.SetAttributes(
		attribute.String("messaging.system", "nasaoj-worker"),
		attribute.Int64("submission.id", job.SubId),
		attribute.String("stage.name", job.StageName),
		attribute.String("testcase.name", job.TestName),
		attribute.Int64("job.timeout", job.Timeout))

	if criClient == nil || job.Image == "" {
		if job.Image != "" {
			tracing.Logger.Ctx(ctx).
				Warn("running containerized job outside of container")
		}
		RunShellJob(ctx, retCh, job, cancelRegistry)
		return
	}
	RunContainerJob(ctx, retCh, job, cancelRegistry, criClient, workerBinary)
}
