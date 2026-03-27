package workflows

import (
	"context"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

func PutJob(
	client *resty.Client,
	pctx context.Context,
	retCh chan error,
	jobch chan models.JobWithFile,
) {
	job := <-jobch

	ctx, span := tracing.Tracer.Start(pctx, "worker.PutJob")
	defer span.End()

	span.SetAttributes(
		attribute.String("messaging.system", "nasaoj-worker"),
		attribute.Int64("submission.id", job.SubId),
		attribute.String("stage.name", job.StageName),
		attribute.String("testcase.name", job.TestName),
		attribute.Int64("job.status", job.Status))

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
