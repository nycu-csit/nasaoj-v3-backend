package deletes

import (
	"fmt"
	"net/url"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
)

var submissionCmd = &cobra.Command{
	Use:   "submission [submission_id]",
	Short: "Delete submission",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, ok := cmd.Context().Value(types.RestyClient("")).(*resty.Client)
		if !ok {
			return fmt.Errorf("resty client not found")
		}

		submissionURL, err := url.JoinPath(
			viper.GetString("endpoint"), "/api/admin/submission")
		if err != nil {
			return fmt.Errorf("url construction failed, %w", err)
		}

		resp, err := client.R().
			SetQueryParam("subId", args[0]).
			Delete(submissionURL)
		if err != nil {
			return fmt.Errorf("DELETE %s failed, %w", submissionURL, err)
		}

		if !resp.IsSuccess() {
			return fmt.Errorf("%v\n", resp.Error())
		}
		fmt.Printf("%v\n", string(resp.Body()[:]))
		return nil
	},
}
