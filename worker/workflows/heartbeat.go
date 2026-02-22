package workflows

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"go.uber.org/zap"
)

type heartbeatRequest struct {
	Tags []string `json:"tags"`
}

func Heartbeat(client *resty.Client, ctx context.Context) {
	tickCh := time.Tick(time.Duration(
		viper.GetInt64("heartbeat-sec")) * time.Second)

	tags := strings.Split(viper.GetString("tags"), ",")

	sendHeartbeat := func() {
		resp, err := client.R().
			SetBody(heartbeatRequest{Tags: tags}).
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
		}
	}

	sendHeartbeat()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tickCh:
			sendHeartbeat()
		}
	}
}
