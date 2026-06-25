package workflows

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

type heartbeatRequest struct {
	Tags        []string `json:"tags"`
	Submissions []int64  `json:"submissions"`
}

type heartbeatResponse struct {
	CancelSubmissions []int64 `json:"cancelSubmissions"`
}

func Heartbeat(
	client *resty.Client,
	pctx context.Context,
	subCancel *CancelRegistry,
	active *ActiveSubs,
) {
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
			SetBody(heartbeatRequest{Tags: tags, Submissions: active.List()}).
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

		// Abort whole submissions the backend cancelled. Cancelling a
		// submission's context cancels all of its in-flight jobs too.
		if len(result.CancelSubmissions) > 0 {
			tracing.Logger.Ctx(ctx).
				Info("cancelling submissions",
					zap.Int64s("subIds", result.CancelSubmissions))
			subKeys := make([]string, len(result.CancelSubmissions))
			for i, id := range result.CancelSubmissions {
				subKeys[i] = strconv.FormatInt(id, 10)
			}
			subCancel.CancelAll(subKeys)
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
