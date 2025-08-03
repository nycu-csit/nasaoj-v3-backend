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

var problemCmd = &cobra.Command{
	Use:   "problem [problem_sha1]",
	Short: "Enable problem",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, ok := cmd.Context().Value(types.RestyClient("")).(*resty.Client)
		if !ok {
			return fmt.Errorf("resty client not found")
		}

		url, err := url.JoinPath(
			viper.GetString("endpoint"), "/api/admin/problem-state")
		if err != nil {
			return fmt.Errorf("url construction failed, %w", err)
		}

		resp, err := client.R().
			SetBody(apiHandlers.PutProblemStateInput{
				ProblemSHA1: args[0],
				IsEnabled:   true,
			}).
			Put(url)
		if err != nil {
			return fmt.Errorf("PUT %s failed, %w", url, err)
		}

		if !resp.IsSuccess() {
			return fmt.Errorf("%v\n", resp.Error())
		}
		fmt.Printf("%v\n", string(resp.Body()[:]))
		return nil
	},
}
