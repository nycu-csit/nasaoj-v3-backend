package deletes

import (
	"io"
	"os"
	"strconv"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

var userCmd = &cobra.Command{
	Use:   "user [user_csId]",
	Short: "Delete user",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		resp := utils.RestyBuilder(ctx, "/api/admin/user",
			func(r *resty.Request) *resty.Request {
				r.Method = resty.MethodDelete
				return r.
					SetQueryParam("csId", args[0]).
					SetQueryParam("prune",
						strconv.FormatBool(viper.GetBool("prune")))
			})

		io.WriteString(os.Stdout, string(resp.Body()[:]))
	},
}

func init() {
	userCmd.Flags().
		Bool("prune", false, "Also delete related submissions")
}
