package submit

import (
	"io"
	"os"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

var problemCmd = &cobra.Command{
	Use:   "problem [problem_sha1]",
	Short: "Submit problem",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		var output api.SubmitOutput
		utils.RestyBuilder(ctx, "/api/admin/submit",
			func(r *resty.Request) *resty.Request {
				r.Method = resty.MethodPost
				return r.
					SetBody(api.AdminSubmitInput{
						AsCsId: viper.GetString("csid"),
						SubmitInput: api.SubmitInput{
							RefSHA1: args[0],
							Type:    models.RefTypeProblem,
						},
					}).
					SetResult(&output).
					ForceContentType("application/json")
			})

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
