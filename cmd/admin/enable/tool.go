package enable

import (
	"io"
	"os"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

var toolCmd = &cobra.Command{
	Use:   "tool [tool_sha1_prefix]",
	Short: "Enable tool",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		resp := utils.RestyBuilder(ctx, "/api/admin/tool/state",
			func(r *resty.Request) *resty.Request {
				r.Method = resty.MethodPut
				return r.
					SetBody(api.PutToolStateInput{
						ToolSHA1:  args[0],
						IsEnabled: true,
					})
			})

		io.WriteString(os.Stdout, string(resp.Body()[:]))
	},
}
