package get

import (
	"fmt"
	"net/url"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"gopkg.in/yaml.v2"
)

var userCmd = &cobra.Command{
	Use:   "user [user_CsId]",
	Short: "Get User and Resources",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, ok := cmd.Context().Value(types.RestyClient("")).(*resty.Client)
		if !ok {
			return fmt.Errorf("resty client not found")
		}

		url, err := url.JoinPath(
			viper.GetString("endpoint"), "/api/admin/user")
		if err != nil {
			return fmt.Errorf("url construction failed, %w", err)
		}

		var res models.UserResource
		resp, err := client.R().
			SetResult(&res).
			ForceContentType("application/json").
			SetQueryString("csId=" + args[0]).
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
