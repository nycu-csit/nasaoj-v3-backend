package gen

import (
	"fmt"
	"net/url"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	apiHandlers "gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
)

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Generate Access Token",
}

func tokenCmdHandler(
	tokenType string,
) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		client, ok := cmd.Context().Value(types.RestyClient("")).(*resty.Client)
		if !ok {
			return fmt.Errorf("resty client not found")
		}

		url, err := url.JoinPath(
			viper.GetString("endpoint"), "/api/admin/token")
		if err != nil {
			return fmt.Errorf("url construction failed, %w", err)
		}

		var res apiHandlers.PostTokenOutput
		resp, err := client.R().
			SetResult(&res).
			ForceContentType("application/json").
			SetBody(apiHandlers.PostTokenInput{
				TokenType: tokenType,
			}).
			Post(url)
		if err != nil {
			return fmt.Errorf("POST %s failed, %w", url, err)
		}

		if !resp.IsSuccess() {
			return fmt.Errorf("%v\n", resp.Error())
		}

		fmt.Printf("Token: %s\n", res.Token)
		return nil
	}
}

func init() {
	tokenCmd.AddCommand(&cobra.Command{
		Use:   "admin",
		Short: "Generate Admin Access Token",
		RunE:  tokenCmdHandler("admin"),
	})
	tokenCmd.AddCommand(&cobra.Command{
		Use:   "worker",
		Short: "Generate Worker Access Token",
		RunE:  tokenCmdHandler("worker"),
	})
}
