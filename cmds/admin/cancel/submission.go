package cancel

import (
	"io"
	"os"
	"strconv"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

var submissionCmd = &cobra.Command{
	Use:   "submission [submission_id]",
	Short: "Cancel submission",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		subId, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			panic(err)
		}
		resp := utils.RestyBuilder(ctx, "/api/admin/submission/cancel",
			func(r *resty.Request) *resty.Request {
				r.Method = resty.MethodPut
				return r.
					SetBody(api.PutCancelSubmissionInput{
						SubId: subId,
					})
			})

		io.WriteString(os.Stdout, string(resp.Body()[:]))
	},
}
