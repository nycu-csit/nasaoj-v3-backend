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

var resultsCmd = &cobra.Command{
	Use:   "results [submission_id]",
	Short: "Get results",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		var output api.ResultOutput
		reqFunc := func(r *resty.Request) *resty.Request {
			return r.
				SetQueryParam("subId", args[0]).
				SetResult(&output).
				ForceContentType("application/json")
		}
		utils.RestyBuilder(cmd,
			"/api/admin/results", reqFunc)

		b, err := yaml.Marshal(output)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to marshal output to yaml",
					zap.Error(err))
			panic(err)
		}
		io.WriteString(os.Stdout, string(b[:]))
	},
}
