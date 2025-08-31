package disable

import (
	"io"
	"os"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

var problemCmd = &cobra.Command{
	Use:   "problem [problem_sha1]",
	Short: "Disable problem",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		resp := utils.RestyBuilder(cmd, "/api/admin/problem/state",
			func(r *resty.Request) *resty.Request {
				r.Method = resty.MethodPut
				return r.
					SetBody(api.PutProblemStateInput{
						ProblemSHA1: args[0],
						IsEnabled:   false,
					})
			})

		io.WriteString(os.Stdout, string(resp.Body()[:]))
	},
}
