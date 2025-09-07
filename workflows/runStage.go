package workflows

import (
	"bytes"
	"context"
	"time"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.uber.org/zap"
)

func RunStage(
	ctx context.Context,
	jobQue interfaces.JobQueue,
	db interfaces.Database,
	s3 interfaces.ObjectStorage,
	problems interfaces.ProblemStorage,
	contents interfaces.ContentFileStorage,
	tokens interfaces.TokenStorage,

	submission models.Submission,
	stageName string,
) error {
	problem, err := problems.GetProblem(ctx, submission.ProblemSHA1)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("problem not exist",
				zap.String("problem.sha1", submission.ProblemSHA1),
				zap.Error(err))
		return err
	}
	if err := contents.HydrateProblem(ctx, &problem); err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to hydrate problem",
				zap.Error(err))
		return err
	}

	stage, ok := problem.Stages[stageName]
	if !ok {
		tracing.Logger.Ctx(ctx).
			Error("stage not exist",
				zap.String("stage", stageName))
		return types.StageNotExistError
	}

	tracing.Logger.Ctx(ctx).
		Info("run stage",
			zap.String("problem.name", problem.Name),
			zap.String("problem.sha1", submission.ProblemSHA1),
			zap.String("stage.name", stageName))

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

		duration := time.Duration(testcase.HardTL) * time.Second
		deadline := time.Now().Add(duration).Unix()
		jobToken, err := tokens.GenerateToken(
			ctx, models.NewJobPrivs(submission.CsId), duration)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to generate token",
					zap.Error(err))
			return err
		}
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

		if err := db.UpdateJob(ctx, &job); err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to update job status",
					zap.Error(err))
			return err
		}

		args := utils.ArgsOverlay(
			submission.Args,
			problem.Args,
			stage.Args,
			testcase.Args,
			map[string]string{
				"jobToken":  jobToken,
				"jobInput":  job.Id() + ".input",
				"jobScript": job.Id() + ".bash",
			},
		)

		var tarbuf bytes.Buffer
		if err = utils.WriteTar(ctx, &tarbuf, contents, args); err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to write to tar",
					zap.Error(err))
			return err
		}

		err = s3.PutFile(ctx, job.Filename(), tarbuf.Bytes())
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to put file",
					zap.String("file.name", job.Filename()),
					zap.Error(err))
			return err
		}

		err = jobQue.PutJob(ctx, job, problem.Tag)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to put job to queue",
					zap.Error(err))
			return err
		}
	}

	return nil
}
