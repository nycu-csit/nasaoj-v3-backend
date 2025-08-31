package deletes

import (
	"io"
	"os"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

var userCmd = &cobra.Command{
	Use:   "user [user_csId]",
	Short: "Delete user",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		resp := utils.RestyBuilder(cmd, "/api/admin/user",
			func(r *resty.Request) *resty.Request {
				r.Method = resty.MethodDelete
				return r.
					SetQueryParam("csId", args[0])
			})

		io.WriteString(os.Stdout, string(resp.Body()[:]))
	},
}
