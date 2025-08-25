package imports

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

func validateInput(
	ctx context.Context,
	inputs []models.UserResource,
) error {
	for _, input := range inputs {
		if input.CsId == "" {
			tracing.Logger.Ctx(ctx).
				Error("csid cannot be empty")
			return types.WrongFormatError
		}
	}
	return nil
}

var usersCmd = &cobra.Command{
	Use:   "users",
	Short: "Import users, read from stdin",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		client, ok := cmd.Context().Value(types.RestyClient("")).(*resty.Client)
		if !ok {
			tracing.Logger.Ctx(ctx).
				Panic("resty client not exist")
			panic(types.ContextNotExistError)
		}

		usersURL, err := url.JoinPath(
			viper.GetString("endpoint"), "/api/admin/users")
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to construct url",
					zap.Error(err))
			panic(err)
		}

		input, err := utils.ReadStdin(ctx, "---")
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to read from stdin",
					zap.Error(err))
			panic(err)
		}

		var inputUsers []models.UserResource
		if err := yaml.Unmarshal([]byte(input), &inputUsers); err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to unmarshal yaml to user resources",
					zap.Error(err))
			panic(err)
		}

		if err := validateInput(ctx, inputUsers); err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to validate input",
					zap.Error(err))
			panic(err)
		}

		b, err := json.Marshal(inputUsers)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to marshal user resources to json",
					zap.Error(err))
			panic(err)
		}

		resp, err := client.R().
			SetBody(string(b[:])).
			Put(usersURL)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to put user resources",
					zap.String("url", usersURL),
					zap.Error(err))
			panic(err)
		}

		if !resp.IsSuccess() {
			tracing.Logger.Ctx(ctx).
				Panic("resp.IsSuccess failed")
			panic(types.HTTPRequestFailedError)
		}
		fmt.Printf("%s", string(resp.Body()[:]))
	},
}
