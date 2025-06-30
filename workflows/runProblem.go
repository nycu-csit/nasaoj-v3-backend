package workflows

import (
	"context"
	"fmt"
	"os"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

// Do Topological Sorting to stages
func RunProblem(
	ctx context.Context,
	jobQue interfaces.JobQueue,
	db interfaces.Database,
	s3 interfaces.ObjectStorage,
	problems interfaces.ProblemStorage,

	submission models.Submission,
) error {
	problem, err := problems.GetProblem(ctx, submission.ProblemSHA1)
	if err != nil {
		return fmt.Errorf("problem with id '%d' not exist, %w",
			submission.ProblemId, err)
	}

	fmt.Fprintf(os.Stderr, "run problem '%s'\n", problem.Name)

	// Calc dependency count (indegree of the stages)
	stageDeps := make(map[string]int)
	for stageName, stage := range problem.Stages {
		err := jobQue.SetJobCnt(ctx,
			fmt.Sprintf("%d-%s", submission.Id, stageName), len(stage.Testcases))
		if err != nil {
			return err
		}

		for _, next := range stage.Before {
			stageDeps[next] += 1
		}
	}

	// Run stages without dependency
	for stageName := range problem.Stages {
		if _, exist := stageDeps[stageName]; !exist {
			err := RunStage(ctx, jobQue, db, s3, problems, submission, stageName)
			if err != nil {
				return err
			}
		} else {
			err := jobQue.SetDepCnt(ctx,
				fmt.Sprintf("%d-%s", submission.Id, stageName), stageDeps[stageName])
			if err != nil {
				return err
			}
		}
	}
	return nil
}
