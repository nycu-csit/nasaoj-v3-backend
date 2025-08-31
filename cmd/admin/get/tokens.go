package get

import (
	"io"
	"os"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

var tokensCmd = &cobra.Command{
	Use:   "tokens",
	Short: "Get tokens",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		var res []api.GetTokensOutput
		utils.RestyBuilder(cmd, "/api/admin/tokens",
			func(r *resty.Request) *resty.Request {
				return r.
					SetResult(&res).
					ForceContentType("application/json")
			})

		b, err := yaml.Marshal(res)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to marshal tokens output",
					zap.Error(err))
			panic(err)
		}
		io.WriteString(os.Stdout, string(b[:]))
	},
}
