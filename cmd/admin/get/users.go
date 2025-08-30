package get

import (
	"io"
	"os"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

var usersCmd = &cobra.Command{
	Use:   "users",
	Short: "Get users with args",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		var users []models.UserResource
		reqFunc := func(r *resty.Request) *resty.Request {
			return r.
				SetResult(&users).
				ForceContentType("application/json")
		}
		utils.RestyBuilder(cmd,
			"/api/admin/users", reqFunc)

		b, err := yaml.Marshal(users)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to marshal users to yaml",
					zap.Error(err))
			panic(err)
		}
		io.WriteString(os.Stdout, string(b[:]))
	},
}
