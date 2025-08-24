package get

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
	"gopkg.in/yaml.v2"
)

var tokensCmd = &cobra.Command{
	Use:   "tokens",
	Short: "Get tokens",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		client, ok := cmd.Context().Value(types.RestyClient("")).(*resty.Client)
		if !ok {
			tracing.Logger.Ctx(ctx).
				Panic("resty client not exist")
			panic(types.ContextNotExistError)
		}

		tokensURL, err := url.JoinPath(
			viper.GetString("endpoint"), "/api/admin/tokens")
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to construct url",
					zap.Error(err))
			panic(err)
		}

		var res []apiHandlers.GetTokensOutput
		resp, err := client.R().
			SetResult(&res).
			ForceContentType("application/json").
			Get(tokensURL)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to get tokens",
					zap.String("url", tokensURL),
					zap.Error(err))
			panic(err)
		}

		if !resp.IsSuccess() {
			tracing.Logger.Ctx(ctx).
				Panic("resp.IsSuccess failed")
			panic(types.HTTPRequestFailedError)
		}

		b, err := yaml.Marshal(res)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to marshal tokens output",
					zap.Error(err))
			panic(err)
		}
		fmt.Printf("%s", string(b[:]))
	},
}
