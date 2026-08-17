package get

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.uber.org/zap"
	"go.yaml.in/yaml/v3"
)

type tokenResult struct {
	SHA256      string `yaml:"sha256"`
	Description string `yaml:"description"`
	CreatedAt   string `yaml:"createdAt"`
	IsEnabled   bool   `yaml:"isEnabled"`
	Privilege   string `yaml:"privilege"`
	UserId      int64  `yaml:"userId,omitempty"`
}

func scopeName(scope int) string {
	switch scope {
	case models.ScopeAll:
		return "all"
	case models.ScopeSelf:
		return "self"
	default:
		return "no"
	}
}

// privilegeSummary renders Privileges as a compact, human-readable string,
// e.g. "read:all write:all" or "userArgs:all schedule:all" for a worker.
func privilegeSummary(p models.Privileges) string {
	var parts []string
	if p.CanAPIRead != models.ScopeNo {
		parts = append(parts, "read:"+scopeName(p.CanAPIRead))
	}
	if p.CanAPIWrite != models.ScopeNo {
		parts = append(parts, "write:"+scopeName(p.CanAPIWrite))
	}
	if p.CanUserArgsWrite != models.ScopeNo {
		parts = append(parts, "userArgs:"+scopeName(p.CanUserArgsWrite))
	}
	if p.CanScheduleAPI != models.ScopeNo {
		parts = append(parts, "schedule:"+scopeName(p.CanScheduleAPI))
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, " ")
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
			privilege := privilegeSummary(token.Token.Privs)
			if token.Token.UserId != 0 {
				// The token's Privs are just a snapshot from issuance; the
				// user's live privileges/namespaces are what's actually
				// enforced on each request.
				privilege += " (live, resolved from user)"
			}

			ret = append(ret, tokenResult{
				SHA256:      token.SHA256,
				Description: token.Token.Description,
				CreatedAt:   time.Unix(token.Token.CreatedAt, 0).String(),
				IsEnabled:   token.Token.IsEnabled,
				Privilege:   privilege,
				UserId:      token.Token.UserId,
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
