package namespace

import (
	"io"
	"os"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.uber.org/zap"
	"go.yaml.in/yaml/v3"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List namespaces",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		var namespaces []models.Namespace
		utils.RestyBuilder(ctx, "/api/admin/namespaces",
			func(r *resty.Request) *resty.Request {
				return r.
					SetResult(&namespaces).
					ForceContentType("application/json")
			})

		b, err := yaml.Marshal(namespaces)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to marshal namespaces to yaml",
					zap.Error(err))
			panic(err)
		}
		io.WriteString(os.Stdout, string(b[:]))
	},
}
