package worker

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func FetchJob(
	client *resty.Client,
	pctx context.Context,
	retCh chan error,
	jobIn chan models.JobWithFile,
) {
	ctx, cancel := context.WithCancel(pctx)
	defer cancel()

	var job models.Job
	resp, err := client.R().
		SetContext(ctx).
		SetResult(&job).
		ForceContentType("application/json").
		SetQueryParam("tags", viper.GetString("tags")).
		Get("/schedule/job")

	if resp.StatusCode() == http.StatusNoContent {
		tracing.Logger.Ctx(ctx).
			Warn("job not exist")
		retCh <- types.JobNotExistError
		return
	}

	if !resp.IsSuccess() {
		tracing.Logger.Ctx(ctx).
			Error("failed to get job",
				zap.Error(err))
		retCh <- types.HTTPRequestFailedError
		return
	}

	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.Int64("submission.id", job.SubId),
		attribute.String("stage.name", job.StageName),
		attribute.String("testcase.name", job.TestName))

	refFilePath := path.Join(viper.GetString("running-dir"), job.RefFilename())
	_, err = os.Stat(refFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			refFileURL, err := url.Parse(job.RefFileURL)
			if err != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to parse file url",
						zap.String("url", job.JobFileURL),
						zap.Error(err))
				retCh <- types.WrongFormatError
				return
			}
			resp, err = resty.New().R().
				SetContext(ctx).
				SetQueryString(refFileURL.RawQuery).
				Get(strings.Split(refFileURL.String(), "?")[0])
			if err != nil || !resp.IsSuccess() {
				tracing.Logger.Ctx(ctx).
					Error("failed to get file",
						zap.String("url", refFileURL.String()),
						zap.Error(err))
				if err != nil {
					retCh <- err
				} else {
					retCh <- types.HTTPRequestFailedError
				}
				return
			}

			if err := os.WriteFile(refFilePath, resp.Body(), 0644); err != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to write ref file",
						zap.Error(err))
				retCh <- err
				return
			}
		} else {
			tracing.Logger.Ctx(ctx).
				Error("failed to stat ref file",
					zap.String("refFile.path", refFilePath),
					zap.Error(err))
			retCh <- err
			return
		}
	}

	jobFileURL, err := url.Parse(job.JobFileURL)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to parse file url",
				zap.String("url", job.JobFileURL),
				zap.Error(err))
		retCh <- types.WrongFormatError
		return
	}
	resp, err = resty.New().R().
		SetContext(ctx).
		SetQueryString(jobFileURL.RawQuery).
		Get(strings.Split(jobFileURL.String(), "?")[0])
	if err != nil || !resp.IsSuccess() {
		tracing.Logger.Ctx(ctx).
			Error("failed to get file",
				zap.String("url", jobFileURL.String()),
				zap.Error(err))
		if err != nil {
			retCh <- err
		} else {
			retCh <- types.HTTPRequestFailedError
		}
		return
	}

	jobIn <- models.JobWithFile{
		Job:    job,
		Tarbuf: resp.Body(),
	}
	retCh <- nil
}
