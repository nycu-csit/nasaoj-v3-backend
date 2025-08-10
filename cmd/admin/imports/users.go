package imports

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"gopkg.in/yaml.v2"
)

func validateInput(inputs []models.UserResource) error {
	for _, input := range inputs {
		if input.CsId == "" {
			return fmt.Errorf("csId cannot be empty")
		}
	}
	return nil
}

var usersCmd = &cobra.Command{
	Use:   "users",
	Short: "Import users, read from stdin",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, ok := cmd.Context().Value(types.RestyClient("")).(*resty.Client)
		if !ok {
			return fmt.Errorf("resty client not found")
		}

		url, err := url.JoinPath(
			viper.GetString("endpoint"), "/api/admin/users")
		if err != nil {
			return fmt.Errorf("url construction failed, %w", err)
		}

		input, err := utils.ReadStdin("---")
		if err != nil {
			return fmt.Errorf("read content failed, %w", err)
		}

		var inputUsers []models.UserResource
		if err := yaml.Unmarshal([]byte(input), &inputUsers); err != nil {
			return fmt.Errorf("yaml unmarshal failed, %w", err)
		}

		if err := validateInput(inputUsers); err != nil {
			return fmt.Errorf("input validate failed, %w", err)
		}

		b, err := json.Marshal(inputUsers)
		if err != nil {
			return fmt.Errorf("json marshal failed, %w", err)
		}

		resp, err := client.R().
			SetBody(string(b[:])).
			Put(url)
		if err != nil {
			return fmt.Errorf("PUT %s failed, %w", url, err)
		}

		if !resp.IsSuccess() {
			return fmt.Errorf("%v\n", resp.Error())
		}
		fmt.Printf("%s", string(resp.Body()[:]))
		return nil
	},
}
