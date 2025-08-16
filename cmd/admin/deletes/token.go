package deletes

import (
	"fmt"
	"net/url"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
)

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Invalidate token",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, ok := cmd.Context().Value(types.RestyClient("")).(*resty.Client)
		if !ok {
			return fmt.Errorf("resty client not found")
		}

		url, err := url.JoinPath(
			viper.GetString("endpoint"), "/api/admin/token")
		if err != nil {
			return fmt.Errorf("url construction failed, %w", err)
		}

		resp, err := client.R().
			SetQueryString("token=" + args[0]).
			Delete(url)
		if err != nil {
			return fmt.Errorf("DELETE %s failed, %w", url, err)
		}

		if !resp.IsSuccess() {
			return fmt.Errorf("%v\n", resp.Error())
		}
		fmt.Printf("%v\n", string(resp.Body()[:]))
		return nil
	},
}
