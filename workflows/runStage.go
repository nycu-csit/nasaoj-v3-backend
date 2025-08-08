package workflows

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

func RunStage(
	ctx context.Context,
	jobQue interfaces.JobQueue,
	db interfaces.Database,
	s3 interfaces.ObjectStorage,
	problems interfaces.ProblemStorage,

	submission models.Submission,
	stageName string,
) error {
	problem, err := problems.GetProblem(ctx, submission.ProblemSHA1)
	if err != nil {
		return fmt.Errorf("problem with sha1 '%s' not exist",
			submission.ProblemSHA1)
	}
	stage, ok := problem.Stages[stageName]
	if !ok {
		return fmt.Errorf("stage '%s' not exist", stageName)
	}

	fmt.Fprintf(os.Stderr, "run stage '%s' from '%s'\n", stageName, problem.Name)

	for _, testcase := range stage.Testcases {
		contents := []models.Content{
			problem.PreScript,
			stage.PreScript,
			testcase.Content,
			stage.PostScript,
			problem.PostScript,
		}

		jobScript := ""
		for _, content := range contents {
			for _, script := range content.Scripts {
				jobScript += script + "\n"
			}
		}

		deadline := time.Now().
			Add(time.Duration(testcase.HardTL) * time.Second).Unix()
		job := models.Job{
			SubId:     submission.Id,
			StageName: stageName,
			TestName:  testcase.Name,

			Status:   models.StatusPending,
			Script:   jobScript,
			Deadline: deadline,
			HardTL:   testcase.HardTL,
			SoftTL:   testcase.SoftTL,
		}

		err := db.UpdateJob(ctx, &job)
		if err != nil {
			return fmt.Errorf("update job status failed, %w", err)
		}

		args := utils.ArgsOverlay(
			submission.Args,
			problem.Args,
			stage.Args,
			testcase.Args,
		)

		var tarbuf bytes.Buffer
		if err = utils.WriteTar(ctx, problems, &tarbuf, contents, args); err != nil {
			return fmt.Errorf("write to tar failed, %w", err)
		}

		err = s3.PutFile(ctx, job.Filename(), tarbuf.Bytes())
		if err != nil {
			return fmt.Errorf("put file '%s' failed, %w", job.Filename(), err)
		}

		timeout := time.Duration(job.HardTL) * time.Second
		job.FileURL, err = s3.GetFileURL(ctx, job.Filename(), timeout)
		if err != nil {
			return fmt.Errorf("get file url for '%s' failed, %w", job.Filename(), err)
		}

		err = jobQue.PutJob(ctx, job, problem.Tag)
		if err != nil {
			return fmt.Errorf("put job to queue failed, %w", err)
		}
	}

	return nil
}
