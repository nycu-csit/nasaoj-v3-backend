package worker

import (
	"context"
	"net/url"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"go.uber.org/zap"
)

func PutJob(
	pctx context.Context,
	retCh chan error,
	jobch chan models.JobWithFile,
) {
	ctx, cancel := context.WithCancel(pctx)
	defer cancel()

	client := resty.New()
	client.SetHeader("Authorization",
		"Bearer "+viper.GetString("access-token"))

	resultURL, err := url.JoinPath(
		viper.GetString("scheduler-url"), "/schedule/result")
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to construct url")
		retCh <- err
		return
	}

	job := <-jobch

	for i := 0; i < 3; i++ {
		resp, err := client.R().
			SetContext(ctx).
			SetBody(job.Job).
			Put(resultURL)

		if err == nil && resp.IsSuccess() {
			retCh <- nil
			return
		}
	}

	tracing.Logger.Ctx(ctx).
		Error("failed to put job",
			zap.Error(err))
	retCh <- err
}
