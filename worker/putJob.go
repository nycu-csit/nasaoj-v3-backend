package worker

import (
	"context"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"go.uber.org/zap"
)

func PutJob(
	client *resty.Client,
	pctx context.Context,
	retCh chan error,
	jobch chan models.JobWithFile,
) {
	ctx, cancel := context.WithCancel(pctx)
	defer cancel()

	job := <-jobch

	resp, err := client.R().
		SetContext(ctx).
		SetBody(job.Job).
		Put("/schedule/result")

	for i := 0; i < viper.GetInt("putjob-retry"); i++ {
		if err == nil && resp.IsSuccess() {
			retCh <- nil
			return
		}

		resp, err = client.R().
			SetContext(ctx).
			SetBody(job.Job).
			Put("/schedule/result")
	}
	if err == nil && resp.IsSuccess() {
		retCh <- nil
		return
	}

	tracing.Logger.Ctx(ctx).
		Error("failed to put job",
			zap.Error(err))
	retCh <- err
}
