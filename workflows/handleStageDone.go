package workflows

import (
	"context"
	"fmt"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

func HandleStageDone(
	ctx context.Context,
	db interfaces.Database,
	jobQue interfaces.JobQueue,
	s3 interfaces.ObjectStorage,
	scoreCache interfaces.ScoreCache,

	problems models.ProblemRepository,
	testcases *models.TestcaseRepository,

	job models.Job,
) error {
	jobCnt, err := jobQue.JobDone(ctx, job.Stage())
	if err != nil {
		return fmt.Errorf("job queue set done failed, %w", err)
	}

	if jobCnt > 0 {
		return nil
	}

	submission, err := db.GetSubmission(ctx, job.SubId)
	if err != nil {
		return fmt.Errorf("get submission failed, %w", err)
	}

	problem, ok := problems[submission.ProblemId]
	if !ok {
		return fmt.Errorf("problem with id '%d' not exist", submission.ProblemId)
	}
	stage, ok := problem.Stages[job.StageName]
	if !ok {
		return fmt.Errorf("stage '%s' not exist", job.StageName)
	}

	err = db.IncreaseScore(ctx, job.SubId, job.StageName, stage.Score)
	if err != nil {
		return fmt.Errorf("score put to db failed, %w", err)
	}
	err = scoreCache.Put(ctx, problem, job.StageName,
		submission.CsId, stage.Score)
	if err != nil {
		return fmt.Errorf("score put to cache failed, %w", err)
	}

	for _, next := range stage.Before {
		jobdep, err := jobQue.StageDone(ctx, fmt.Sprintf("%d-%s", job.SubId, next))
		if err != nil {
			return fmt.Errorf("jobdep decrease failed, %w", err)
		}

		if jobdep <= 0 {
			err = RunStage(ctx, jobQue, db, s3, problems, testcases, submission, next)
			if err != nil {
				return fmt.Errorf("run stage failed, %w", err)
			}
		}
	}

	return nil
}
