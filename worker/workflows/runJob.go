package workflows

import (
	"context"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/worker/cri"
)

func RunJob(
	pctx context.Context,
	retCh chan error,
	job *models.JobWithFile,
	cancelRegistry *CancelRegistry,
	criClient *cri.Client,
	workerBinary string,
) {
	if criClient == nil || job.Image == "" {
		if job.Image != "" {
			tracing.Logger.Ctx(pctx).
				Warn("running containerized job outside of container")
		}
		RunShellJob(pctx, retCh, job, cancelRegistry)
		return
	}
	RunContainerJob(pctx, retCh, job, cancelRegistry, criClient, workerBinary)
}
