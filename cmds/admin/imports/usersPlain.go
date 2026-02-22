package imports

import (
	"encoding/json"
	"io"
	"os"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

var usersPlainCmd = &cobra.Command{
	Use:   "plain",
	Short: "Import users",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		stdin, err := utils.ReadStdin(ctx, `---
# Example
# - username: 'phkoan'
#   password: 'verysecurepassword'
#   args:
#     poop: abc
`)
		if err != nil {
			panic(err)
		}

		var inputs []api.PutUsersPlainInput
		if err := yaml.Unmarshal([]byte(stdin), &inputs); err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to unmarshal yaml",
					zap.Error(err))
			panic(err)
		}

		for _, input := range inputs {
			if input.Username == "" {
				tracing.Logger.Ctx(ctx).
					Error("username cannot be empty")
				panic(types.WrongFormatError)
			}
			if input.Password == "" {
				tracing.Logger.Ctx(ctx).
					Error("password cannot be empty")
				panic(types.WrongFormatError)
			}
		}

		b, err := json.Marshal(inputs)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to marshal to json",
					zap.Error(err))
			panic(err)
		}

		resp := utils.RestyBuilder(ctx, "/api/admin/users/plain",
			func(r *resty.Request) *resty.Request {
				r.Method = resty.MethodPut
				return r.
					SetBody(string(b[:]))
			})
		io.WriteString(os.Stdout, string(resp.Body()[:]))
	},
}
