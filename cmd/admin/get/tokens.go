package get

import (
	"fmt"
	"net/url"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	apiHandlers "gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"gopkg.in/yaml.v2"
)

var tokensCmd = &cobra.Command{
	Use:   "tokens",
	Short: "Get tokens",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, ok := cmd.Context().Value(types.RestyClient("")).(*resty.Client)
		if !ok {
			return fmt.Errorf("resty client not found")
		}

		url, err := url.JoinPath(
			viper.GetString("endpoint"), "/api/admin/tokens")
		if err != nil {
			return fmt.Errorf("url construction failed, %w", err)
		}

		var res []apiHandlers.GetTokensOutput
		resp, err := client.R().
			SetResult(&res).
			ForceContentType("application/json").
			Get(url)
		if err != nil {
			return fmt.Errorf("GET %s failed, %w", url, err)
		}

		if !resp.IsSuccess() {
			return fmt.Errorf("%v\n", resp.Error())
		}

		b, err := yaml.Marshal(res)
		if err != nil {
			return fmt.Errorf("yaml marshal failed, %w", err)
		}
		fmt.Printf("%s", string(b[:]))
		return nil

	},
}
