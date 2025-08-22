package deletes

import (
	"fmt"
	"net/url"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
)

var problemCmd = &cobra.Command{
	Use:     "problem [problem_sha1]",
	Short:   "Delete problem and related submissions",
	Aliases: []string{"tool"},
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, ok := cmd.Context().Value(types.RestyClient("")).(*resty.Client)
		if !ok {
			return fmt.Errorf("resty client not found")
		}

		problemURL, err := url.JoinPath(
			viper.GetString("endpoint"), "/api/admin/problem")
		if err != nil {
			return fmt.Errorf("url construction failed, %w", err)
		}

		resp, err := client.R().
			SetQueryParam("problemSHA1", args[0]).
			Delete(problemURL)
		if err != nil {
			return fmt.Errorf("DELETE %s failed, %w", problemURL, err)
		}

		if !resp.IsSuccess() {
			return fmt.Errorf("%v\n", resp.Error())
		}
		fmt.Printf("%v\n", string(resp.Body()[:]))
		return nil
	},
}
