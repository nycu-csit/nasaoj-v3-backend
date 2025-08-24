package worker

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"go.uber.org/zap"
)

func FetchJob(
	pctx context.Context,
	retCh chan error,
	jobIn chan models.JobWithFile,
) {
	ctx, cancel := context.WithCancel(pctx)
	defer cancel()

	client := resty.New()

	jobURL, err := url.JoinPath(
		viper.GetString("scheduler-url"), "/schedule/job")
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to construct url",
				zap.Error(err))
		retCh <- err
		return
	}

	var job models.Job
	resp, err := client.R().
		SetHeader("Authorization",
			"Bearer "+viper.GetString("access-token")).
		SetContext(ctx).
		SetResult(&job).
		ForceContentType("application/json").
		SetQueryParam("tags", viper.GetString("tags")).
		Get(jobURL)

	if resp.StatusCode() == http.StatusNoContent {
		tracing.Logger.Ctx(ctx).
			Error("job not exist")
		retCh <- types.JobNotExistError
		return
	}

	if !resp.IsSuccess() {
		tracing.Logger.Ctx(ctx).
			Error("failed to get job",
				zap.String("url", jobURL))
		retCh <- types.HTTPRequestFailedError
		return
	}

	fileURL, err := url.Parse(job.FileURL)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to parse file url",
				zap.String("url", job.FileURL),
				zap.Error(err))
		retCh <- types.WrongFormatError
		return
	}
	resp, err = client.R().
		SetContext(ctx).
		SetQueryString(fileURL.RawQuery).
		Get(strings.Split(fileURL.String(), "?")[0])

	if err != nil {
		retCh <- err
		return
	}

	if !resp.IsSuccess() {
		tracing.Logger.Ctx(ctx).
			Error("failed to get file",
				zap.String("url", fileURL.String()))
		retCh <- types.HTTPRequestFailedError
		return
	}

	jobIn <- models.JobWithFile{
		Job:    job,
		Tarbuf: resp.Body(),
	}
	retCh <- nil
}
