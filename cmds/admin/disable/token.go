package disable

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

var tokenCmd = &cobra.Command{
	Use:   "token [token]",
	Short: "Disable token",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		// Hash the token to get the key used in storage
		h := sha256.Sum256([]byte(args[0]))
		tokenSHA256 := hex.EncodeToString(h[:])

		resp := utils.RestyBuilder(ctx, "/api/admin/token/state",
			func(r *resty.Request) *resty.Request {
				r.Method = resty.MethodPut
				return r.
					SetBody(api.PutTokenStateInput{
						Token: tokenSHA256,
						State: false,
					})
			})

		io.WriteString(os.Stdout, string(resp.Body()[:]))
	},
}
