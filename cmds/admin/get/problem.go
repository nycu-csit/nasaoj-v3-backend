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
	"go.yaml.in/yaml/v3"
)

var problemDetail bool

var problemCmd = &cobra.Command{
	Use:   "problem [sha1]",
	Short: "Get problem",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		var out any
		if problemDetail {
			out = &api.ProblemDetailOutput{}
		} else {
			out = &api.ProblemOutput{}
		}

		utils.RestyBuilder(ctx, "/api/admin/problem",
			func(r *resty.Request) *resty.Request {
				r = r.
					SetQueryParam("problemSHA1", args[0]).
					SetResult(out).
					ForceContentType("application/json")
				if problemDetail {
					r = r.SetQueryParam("detail", "true")
				}
				return r
			})

		b, err := yaml.Marshal(out)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to marshal output to yaml",
					zap.Error(err))
			panic(err)
		}
		io.WriteString(os.Stdout, string(b[:]))
	},
}

func init() {
	problemCmd.Flags().BoolVarP(&problemDetail, "detail", "d", false,
		"Show the full problem definition (every field, including "+
			"testcase file hashes) instead of the summary")
}
