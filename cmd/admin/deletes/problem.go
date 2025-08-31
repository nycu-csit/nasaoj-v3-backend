package deletes

import (
	"io"
	"os"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

var problemCmd = &cobra.Command{
	Use:   "problem [problem_sha1]",
	Short: "Delete problem and related submissions",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		resp := utils.RestyBuilder(ctx, "/api/admin/problem",
			func(r *resty.Request) *resty.Request {
				r.Method = resty.MethodDelete
				return r.
					SetQueryParam("problemSHA1", args[0])
			})

		io.WriteString(os.Stdout, string(resp.Body()[:]))
	},
}
