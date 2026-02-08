package cancel

import (
	"io"
	"os"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

var submissionCmd = &cobra.Command{
	Use:   "submission [submission_id]",
	Short: "Cancel submission",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		resp := utils.RestyBuilder(ctx, "/api/admin/submission/cancel",
			func(r *resty.Request) *resty.Request {
				r.Method = resty.MethodPut
				return r.
					SetQueryParam("subId", args[0])
			})

		io.WriteString(os.Stdout, string(resp.Body()[:]))
	},
}
