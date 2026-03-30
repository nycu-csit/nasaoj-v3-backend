package tests

import (
	"context"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

const (
	outputMaxRetries    = 50
	outputRetryInterval = 500 * time.Millisecond
)

var RootCmd = &cobra.Command{
	Use:   "test",
	Short: "Test problems or tools",
}

func init() {
	RootCmd.AddCommand(problemCmd)
	RootCmd.AddCommand(toolCmd)
}

func fetchOutput(
	ctx context.Context,
	endpoint, subId, stage, testcase string,
) string {
	for range outputMaxRetries {
		resp := utils.RestyBuilder(ctx, endpoint,
			func(r *resty.Request) *resty.Request {
				return r.
					SetQueryParam("subId", subId).
					SetQueryParam("stage", stage).
					SetQueryParam("testcase", testcase)
			})
		if resp.IsSuccess() {
			return string(resp.Body())
		}
		time.Sleep(outputRetryInterval)
	}
	return "<no-value>"
}
