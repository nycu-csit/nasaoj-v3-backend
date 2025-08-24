package get

import (
	"fmt"
	"net/url"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

var userCmd = &cobra.Command{
	Use:   "user [user_CsId]",
	Short: "Get User and Resources",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		client, ok := cmd.Context().Value(types.RestyClient("")).(*resty.Client)
		if !ok {
			tracing.Logger.Ctx(ctx).
				Panic("resty client not exist")
			panic(types.ContextNotExistError)
		}

		userURL, err := url.JoinPath(
			viper.GetString("endpoint"), "/api/admin/user")
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to construct url",
					zap.Error(err))
			panic(err)
		}

		var res models.UserResource
		resp, err := client.R().
			SetResult(&res).
			ForceContentType("application/json").
			SetQueryString("csId=" + args[0]).
			Get(userURL)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to get user",
					zap.String("url", userURL),
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
				Panic("failed to marshal user resource to yaml",
					zap.Error(err))
			panic(err)
		}
		fmt.Printf("%s", string(b[:]))
	},
}
