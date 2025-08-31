package imports

import (
	"context"
	"encoding/json"
	"io"
	"os"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
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

		input, err := utils.ReadStdin(ctx, `---
# Example
# - csId: '111550001'
#	 args:
#		 poop: abc
`)
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

		resp := utils.RestyBuilder(ctx, "/api/admin/users",
			func(r *resty.Request) *resty.Request {
				r.Method = resty.MethodPut
				return r.
					SetBody(string(b[:]))
			})

		io.WriteString(os.Stdout, string(resp.Body()[:]))
	},
}
