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

var usersCSSSOCmd = &cobra.Command{
	Use:   "cssso",
	Short: "Import users, read from stdin",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		stdin, err := utils.ReadStdin(ctx, `---
# Example
# - csId: '111550001'
#	  args:
#		  poop: abc
`)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to read from stdin",
					zap.Error(err))
			panic(err)
		}

		var inputs []api.PutUsersCSSSOInput
		if err := yaml.Unmarshal([]byte(stdin), &inputs); err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to unmarshal yaml",
					zap.Error(err))
			panic(err)
		}

		for _, input := range inputs {
			if input.CsId == "" {
				tracing.Logger.Ctx(ctx).
					Error("csId cannot be empty")
				panic(types.WrongFormatError)
			}
		}

		b, err := json.Marshal(inputs)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to marshal to json",
					zap.Error(err))
			panic(err)
		}

		resp := utils.RestyBuilder(ctx, "/api/admin/users/cssso",
			func(r *resty.Request) *resty.Request {
				r.Method = resty.MethodPut
				return r.
					SetBody(string(b[:]))
			})

		io.WriteString(os.Stdout, string(resp.Body()[:]))
	},
}
