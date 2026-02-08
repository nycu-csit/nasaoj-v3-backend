package workflows

import (
	"context"
	"fmt"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"go.uber.org/zap"
)

// Do Topological Sorting to stages
func RunProblem(
	ctx context.Context,
	jobQue interfaces.JobQueue,
	db interfaces.Database,
	s3 interfaces.ObjectStorage,
	problems interfaces.ProblemStorage,
	contents interfaces.ContentFileStorage,
	jobs interfaces.JobStorage,
	tokens interfaces.TokenStorage,

	submission models.Submission,
) error {
	problem, err := problems.GetProblem(ctx, submission.ProblemSHA1)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("problem not exist",
				zap.String("problem", submission.ProblemSHA1),
				zap.Error(err))
		return err
	}

	tracing.Logger.Ctx(ctx).
		Info("run problem",
			zap.String("problem.name", problem.Name),
			zap.String("problem.sha1", submission.ProblemSHA1))

	// Calc dependency count (indegree of the stages)
	stageDeps := make(map[string]int)
	for stageName, stage := range problem.Stages {
		err := jobQue.SetJobCnt(ctx,
			fmt.Sprintf("%d-%s", submission.Id, stageName), len(stage.Testcases))
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to set jobCnt",
					zap.Error(err))
			return err
		}

		for _, next := range stage.Before {
			stageDeps[next] += 1
		}
	}

	// Run stages without dependency
	for stageName := range problem.Stages {
		if _, exist := stageDeps[stageName]; !exist {
			err := RunStage(ctx, jobQue, db, s3, problems, contents, jobs, tokens,
				submission, stageName)
			if err != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to run stage",
						zap.String("problem.sha1", submission.ProblemSHA1),
						zap.String("stage.name", stageName),
						zap.Error(err))
				return err
			}
		} else {
			err := jobQue.SetDepCnt(ctx,
				fmt.Sprintf("%d-%s", submission.Id, stageName), stageDeps[stageName])
			if err != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to set depCnt",
						zap.Error(err))
				return err
			}
		}
	}
	return nil
}
