package workflows

import (
	"context"

	"github.com/fatih/color"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
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
	color.Magenta("Worker thread is running\n")
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
			color.Yellow("RunJob return with '%v'\n", ret)
			jobOut <- job
		}
	}
}
