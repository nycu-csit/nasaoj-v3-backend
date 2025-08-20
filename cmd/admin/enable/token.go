package enable

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
	Short: "Enable token",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, ok := cmd.Context().Value(types.RestyClient("")).(*resty.Client)
		if !ok {
			return fmt.Errorf("resty client not found")
		}

		tokenURL, err := url.JoinPath(
			viper.GetString("endpoint"), "/api/admin/token-state")
		if err != nil {
			return fmt.Errorf("url construction failed, %w", err)
		}

		resp, err := client.R().
			SetBody(apiHandlers.PutTokenStateInput{
				Token: args[0],
				State: true,
			}).
			Put(tokenURL)
		if err != nil {
			return fmt.Errorf("PUT %s failed, %w", tokenURL, err)
		}

		if !resp.IsSuccess() {
			return fmt.Errorf("%v\n", resp.Error())
		}
		fmt.Printf("%v\n", string(resp.Body()[:]))
		return nil
	},
}
