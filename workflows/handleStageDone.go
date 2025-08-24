package workflows

import (
	"context"
	"fmt"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"go.uber.org/zap"
)

func HandleStageDone(
	ctx context.Context,
	db interfaces.Database,
	jobQue interfaces.JobQueue,
	s3 interfaces.ObjectStorage,
	problems interfaces.ProblemStorage,
	tokens interfaces.TokenStorage,

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

	submission, err := db.GetSubmission(ctx, job.SubId)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to get submission",
				zap.Error(err))
		return err
	}

	problem, err := problems.GetProblemOrTool(ctx, submission.ProblemSHA1)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("problem not exist",
				zap.String("problem.sha1", submission.ProblemSHA1),
				zap.Error(err))
		return err
	}
	stage, ok := problem.Stages[job.StageName]
	if !ok {
		tracing.Logger.Ctx(ctx).
			Error("stage not exist",
				zap.String("stage.name", job.StageName))
		return types.StageNotExistError
	}

	err = db.IncreaseScore(ctx, job.SubId, job.StageName, stage.Score)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to put score",
				zap.Error(err))
		return err
	}

	for _, next := range stage.Before {
		jobdep, err := jobQue.StageDone(ctx, fmt.Sprintf("%d-%s", job.SubId, next))
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to decrease jobDep",
					zap.Error(err))
			return err
		}

		if jobdep <= 0 {
			err = RunStage(ctx, jobQue, db, s3, problems, tokens, submission, next)
			if err != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to run stage",
						zap.String("stage.name", next),
						zap.Error(err))
				return err
			}
		}
	}

	return nil
}
