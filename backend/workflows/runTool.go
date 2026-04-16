package workflows

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.uber.org/zap"
)

func RunTool(
	ctx context.Context,
	jobQue interfaces.JobQueue,
	db interfaces.Database,
	s3 interfaces.ObjectStorage,
	tools interfaces.ToolStorage,
	contentFiles interfaces.ContentFileStorage,
	jobs interfaces.JobStorage,
	tokens interfaces.TokenStorage,

	submission models.Submission,
) error {
	tool, err := tools.GetTool(ctx, submission.ToolSHA1)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("tool not exist",
				zap.String("tool.sha1", submission.ToolSHA1),
				zap.Error(err))
		return err
	}

	if err := contentFiles.HydrateTool(ctx, &tool); err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to hydrate tool",
				zap.Error(err))
		return err
	}

	tracing.Logger.Ctx(ctx).
		Info("run tool",
			zap.String("tool.name", tool.Name),
			zap.String("tool.sha1", submission.ToolSHA1))

	contents := []models.Content{
		tool.PreScript,
		tool.Testcase.Content,
		tool.PostScript,
	}

	jobScript := ""
	for _, content := range contents {
		for _, script := range content.Scripts {
			jobScript += script + "\n"
		}
	}

	b, err := json.Marshal(submission.InputArgs)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to marshal input args",
				zap.Error(err))
		return err
	}

	jobToken, err := tokens.GenerateToken(
		ctx, models.NewJobPrivs(submission.OwnerId), time.Duration(
			tool.Testcase.Timeout+viper.GetInt64("worker-timeout"))*time.Second)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to generate token",
				zap.Error(err))
		return err
	}
	job := models.Job{
		SubId:     submission.Id,
		StageName: "tool",
		TestName:  tool.Testcase.Name,

		Status:  models.StatusPending,
		Script:  jobScript,
		Input:   string(b[:]),
		Timeout: tool.Testcase.Timeout,
		RefSHA1: submission.ToolSHA1,
	}

	if err := jobs.UpdateJob(ctx, &job, models.StatusBlocked); err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to update job status",
				zap.Error(err))
		return err
	}

	exist, err := s3.ExistFile(ctx, job.RefFilename())
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to check exist of ref file",
				zap.String("refFile.name", job.RefFilename()),
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
		tool.Args,
		tool.Testcase.Args,
		map[string]string{
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

	err = jobQue.PutJob(ctx, job, tool.Tag)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to put job to queue",
				zap.Error(err))
		return err
	}

	return nil
}
