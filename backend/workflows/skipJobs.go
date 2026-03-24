package workflows

import (
	"context"
	"fmt"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
)

func SkipJobs(
	ctx context.Context,
	job *models.Job,
	subs interfaces.SubmissionStorage,
	problems interfaces.ProblemStorage,
	jobs interfaces.JobStorage,
) error {
	submission, err := subs.GetSubmission(ctx, job.SubId)
	if err != nil {
		return types.SubmissionNotExistError
	}

	problem, err := problems.GetProblem(ctx, submission.ProblemSHA1)
	if err != nil {
		return types.ProblemNotExistError
	}

	skips := []string{}
	var traverse func(string)
	traverse = func(name string) {
		skips = append(skips, name)
		for _, next := range problem.Stages[name].Before {
			traverse(next)
		}
	}

	traverse(job.StageName)

	if err := jobs.SkipJobWithStages(ctx, job.SubId, skips); err != nil {
		return fmt.Errorf("failed to skip jobs: %w", err)
	}

	return nil
}
