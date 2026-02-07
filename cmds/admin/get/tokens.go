package get

import (
	"io"
	"os"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type tokenResult struct {
	SHA256    string            `yaml:"sha256"`
	Privs     models.Privileges `yaml:"privs"`
	IsEnabled bool              `yaml:"isEnabled"`
	CreatedAt string            `yaml:"createdAt"`
}

var tokensCmd = &cobra.Command{
	Use:   "tokens",
	Short: "Get tokens",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		var res []api.GetTokensOutput
		utils.RestyBuilder(ctx, "/api/admin/tokens",
			func(r *resty.Request) *resty.Request {
				return r.
					SetResult(&res).
					ForceContentType("application/json")
			})

		var ret []tokenResult
		for _, token := range res {
			ret = append(ret, tokenResult{
				SHA256:    token.SHA256,
				Privs:     token.Token.Privs,
				IsEnabled: token.Token.IsEnabled,
				CreatedAt: time.Unix(token.Token.CreatedAt, 0).String(),
			})
		}

		b, err := yaml.Marshal(ret)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to marshal tokens output",
					zap.Error(err))
			panic(err)
		}
		io.WriteString(os.Stdout, string(b[:]))
	},
}
