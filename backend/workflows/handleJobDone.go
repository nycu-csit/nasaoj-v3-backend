package workflows

import (
	"context"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"go.uber.org/zap"
)

func HandleJobDone(
	ctx context.Context,
	db interfaces.Database,
	jobQue interfaces.JobQueue,
	s3 interfaces.ObjectStorage,
	problems interfaces.ProblemStorage,
	contents interfaces.ContentFileStorage,
	jobs interfaces.JobStorage,
	tokens interfaces.TokenStorage,

	submission models.Submission,
	job models.Job,
) error {
	jobCnt, err := jobQue.JobDone(ctx, job.Stage())
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to set done for job queue",
				zap.Error(err))
		return err
	}

	if jobCnt > 0 {
		return nil
	}

	err = HandleStageDone(ctx, db, jobQue, s3, problems, contents,
		jobs, tokens, submission, job.StageName)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to handle stage done",
				zap.Error(err))
		return err
	}
	return nil
}
