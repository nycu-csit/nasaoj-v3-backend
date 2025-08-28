package workflows

import (
	"context"

	"github.com/fatih/color"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"go.uber.org/zap"
)

type jobInType struct {
	StageName string
	TestName  string
}

func WorkerThread(
	ctx context.Context,
	jobIn chan models.JobWithFile,
	jobOut chan models.JobWithFile,
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
			color.Yellow("%+v\n", jobInType{
				StageName: job.StageName,
				TestName:  job.TestName,
			})
			retCh := make(chan error, 1)
			go RunJob(ctx, retCh, &job)
			ret := <-retCh
			tracing.Logger.Ctx(ctx).
				Info("RunJob returned",
					zap.Error(ret))
			jobOut <- job
		}
	}
}
