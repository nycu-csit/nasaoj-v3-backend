package gen

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

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Generate Access Token",
}

func tokenCmdHandler(
	tokenType string,
) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		client, ok := cmd.Context().Value(types.RestyClient("")).(*resty.Client)
		if !ok {
			tracing.Logger.Ctx(ctx).
				Panic("resty client not exist")
			panic(types.ContextNotExistError)
		}

		tokenURL, err := url.JoinPath(
			viper.GetString("endpoint"), "/api/admin/token")
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to construct url",
					zap.Error(err))
			panic(err)
		}

		var res apiHandlers.PostTokenOutput
		resp, err := client.R().
			SetResult(&res).
			ForceContentType("application/json").
			SetBody(apiHandlers.PostTokenInput{
				TokenType: tokenType,
			}).
			Post(tokenURL)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to post token",
					zap.String("url", tokenURL),
					zap.Error(err))
			panic(err)
		}

		if !resp.IsSuccess() {
			tracing.Logger.Ctx(ctx).
				Panic("resp.IsSuccess failed")
			panic(types.HTTPRequestFailedError)
		}

		fmt.Printf("Token: %s\n", res.Token)
	}
}

func init() {
	tokenCmd.AddCommand(&cobra.Command{
		Use:   "admin",
		Short: "Generate Admin Access Token",
		Run:   tokenCmdHandler("admin"),
	})
	tokenCmd.AddCommand(&cobra.Command{
		Use:   "worker",
		Short: "Generate Worker Access Token",
		Run:   tokenCmdHandler("worker"),
	})
}
