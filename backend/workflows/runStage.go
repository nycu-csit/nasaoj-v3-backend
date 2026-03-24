package workflows

import (
	"bytes"
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/spf13/viper"
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
	jobs interfaces.JobStorage,
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

	// virtual stage
	if len(stage.Testcases) == 0 {
		err := HandleStageDone(ctx, db, jobQue, s3, problems, contents,
			jobs, tokens, submission, stageName)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to handle stage done",
					zap.Error(err))
			return err
		}
		return nil
	}

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

		jobToken, err := tokens.GenerateToken(
			ctx, models.NewJobPrivs(submission.OwnerId), time.Duration(
				2*testcase.Timeout+viper.GetInt64("worker-timeout"))*time.Second)
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

			Status: models.StatusPending,

			Script:        jobScript,
			Timeout:       testcase.Timeout,
			RefSHA1:       submission.ProblemSHA1,
			ResourceGroup: testcase.ResourceGroup,
			Image:         testcase.Image,
		}

		if err := jobs.UpdateJob(ctx, &job); err != nil {
			if errors.Is(err, types.JobAlreadySkippedError) {
				tracing.Logger.Ctx(ctx).
					Warn("job has already skipped")
				return nil
			}
			tracing.Logger.Ctx(ctx).
				Error("failed to update job status",
					zap.Error(err))
			return err
		}

		exist, err := s3.ExistFile(ctx, job.RefFilename())
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to check exist of ref file",
					zap.Error(err))
			return err
		}

		if !exist {
			var tarbuf bytes.Buffer
			if err = utils.WriteRefTar(ctx, &tarbuf, contents); err != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to write to tar",
						zap.Error(err))
				return err
			}

			err = s3.PutFile(ctx, job.RefFilename(), tarbuf.Bytes())
			if err != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to put ref file",
						zap.String("refFile.name", job.RefFilename()),
						zap.Error(err))
				return err
			}
		}

		args := utils.ArgsOverlay(
			submission.Args,
			problem.Args,
			stage.Args,
			testcase.Args,
			map[string]string{
				"subId":     strconv.FormatInt(submission.Id, 10),
				"jobToken":  jobToken,
				"jobInput":  job.Id() + ".input",
				"jobScript": job.Id() + ".bash",
			},
		)

		var tarbuf bytes.Buffer
		if err = utils.WriteJobTar(ctx, &tarbuf, contents, args); err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to write to tar",
					zap.Error(err))
			return err
		}

		err = s3.PutFile(ctx, job.JobFilename(), tarbuf.Bytes())
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to put job file",
					zap.String("jobFile.name", job.JobFilename()),
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
