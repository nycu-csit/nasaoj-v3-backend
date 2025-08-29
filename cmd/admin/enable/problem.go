package enable

import (
	"fmt"
	"net/url"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	apiHandlers "gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"go.uber.org/zap"
)

var problemCmd = &cobra.Command{
	Use:   "problem [problem_sha1]",
	Short: "Enable problem",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		client, ok := cmd.Context().Value(types.RestyClient("")).(*resty.Client)
		if !ok {
			tracing.Logger.Ctx(ctx).
				Panic("resty client not exist")
			panic(types.ContextNotExistError)
		}

		problemURL, err := url.JoinPath(
			viper.GetString("endpoint"), "/api/admin/problem/state")
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to construct url",
					zap.Error(err))
			panic(err)
		}

		resp, err := client.R().
			SetBody(apiHandlers.PutProblemStateInput{
				ProblemSHA1: args[0],
				IsEnabled:   true,
			}).
			Put(problemURL)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to put problem",
					zap.String("url", problemURL),
					zap.Error(err))
			panic(err)
		}

		if !resp.IsSuccess() {
			tracing.Logger.Ctx(ctx).
				Panic("resp.IsSuccess failed")
			panic(types.HTTPRequestFailedError)
		}
		fmt.Printf("%v\n", string(resp.Body()[:]))
	},
}
