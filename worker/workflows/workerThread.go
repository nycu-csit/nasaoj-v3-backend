package workflows

import (
	"context"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/worker/cri"
	"go.uber.org/zap"
)

func WorkerThread(
	ctx context.Context,
	jobIn chan models.JobWithFile,
	jobOut chan models.JobWithFile,
	cancelRegistry *CancelRegistry,
	criClient *cri.Client,
	workerBinary string,
) {
	tracing.Logger.Ctx(ctx).
		Info("worker thread is running")

	for {
		select {
		case <-ctx.Done():
			close(jobOut)
			return
		case job, ok := <-jobIn:
			if !ok {
				close(jobOut)
				return
			}

			tracing.Logger.Ctx(ctx).
				Debug("run job",
					zap.Int64("submission.id", job.SubId),
					zap.String("stage.name", job.StageName),
					zap.String("testcase.name", job.TestName))

			retCh := make(chan error, 1)
			go RunJob(ctx, retCh, &job, cancelRegistry, criClient, workerBinary)
			ret := <-retCh

			tracing.Logger.Ctx(ctx).
				Debug("thread returned",
					zap.Error(ret))
			jobOut <- job
		}
	}
}
