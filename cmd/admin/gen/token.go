package gen

import (
	"fmt"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

func tokenCmdHandler(
	tokenType string,
) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		var res api.PostTokenOutput
		utils.RestyBuilder(cmd, "/api/admin/token",
			func(r *resty.Request) *resty.Request {
				r.Method = resty.MethodPost
				return r.
					SetResult(&res).
					ForceContentType("application/json").
					SetBody(api.PostTokenInput{
						TokenType: tokenType,
					})
			})

		fmt.Printf("Token: %s\n", res.Token)
	}
}

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Generate Access Token",
}

func init() {
	tokenCmd.AddCommand(&cobra.Command{
		Use:   "admin",
		Short: "Generate Admin Access Token",
		Run:   tokenCmdHandler("admin"),
	})
	tokenCmd.AddCommand(&cobra.Command{
		Use:   "worker",
		Short: "Generate Worker Access Token",
		Run:   tokenCmdHandler("worker"),
	})
}
