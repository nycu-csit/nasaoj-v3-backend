package workflows

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

type heartbeatRequest struct {
	Tags []string `json:"tags"`
}

type heartbeatResponse struct {
	CancelJobs []string `json:"cancelJobs"`
}

func Heartbeat(client *resty.Client, pctx context.Context, cancelRegistry *CancelRegistry) {
	tickCh := time.Tick(time.Duration(
		viper.GetInt64("heartbeat-sec")) * time.Second)

	tags := strings.Split(viper.GetString("tags"), ",")

	sendHeartbeat := func() {
		ctx, span := tracing.Tracer.Start(pctx, "worker.Heartbeat")
		defer span.End()
		span.SetAttributes(attribute.String("messaging.system", "nasaoj-worker"))

		var result heartbeatResponse
		resp, err := client.R().
			SetContext(ctx).
			SetBody(heartbeatRequest{Tags: tags}).
			SetResult(&result).
			Post("/schedule/heartbeat")
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Warn("failed to send heartbeat",
					zap.Error(err))
			return
		}
		if resp.StatusCode() != http.StatusOK {
			tracing.Logger.Ctx(ctx).
				Warn("heartbeat returned non-OK status",
					zap.Int("status", resp.StatusCode()))
			return
		}

		// Cancel jobs that the backend requested to be cancelled
		if len(result.CancelJobs) > 0 {
			tracing.Logger.Ctx(ctx).
				Info("cancelling jobs",
					zap.Strings("jobIds", result.CancelJobs))
			cancelRegistry.CancelAll(result.CancelJobs)
		}
	}

	sendHeartbeat()
	for {
		select {
		case <-pctx.Done():
			return
		case <-tickCh:
			sendHeartbeat()
		}
	}
}
