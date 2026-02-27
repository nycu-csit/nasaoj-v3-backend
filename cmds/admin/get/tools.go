package get

import (
	"io"
	"os"
	"sort"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	apiHandlers "gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.uber.org/zap"
	"go.yaml.in/yaml/v3"
)

type toolResult struct {
	ToolId     int64  `json:"toolId" yaml:"toolId"`
	SHA1       string `json:"sha1" yaml:"sha1"`
	ImportedAt string `json:"importedAt" yaml:"importedAt"`

	Name        string `json:"name" yaml:"name"`
	Description string `json:"description" yaml:"description"`
	Tag         string `json:"tag" yaml:"tag"`
	IsEnabled   bool   `json:"isEnabled" yaml:"isEnabled"`
}

var toolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "Get tools",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		var res []apiHandlers.ToolOutput
		utils.RestyBuilder(ctx, "/api/admin/tools",
			func(r *resty.Request) *resty.Request {
				return r.
					SetResult(&res).
					ForceContentType("application/json")
			})

		var ret []toolResult
		for _, tool := range res {
			ret = append(ret, toolResult{
				ToolId:      tool.ToolId,
				SHA1:        tool.SHA1,
				ImportedAt:  time.Unix(tool.ImportedAt, 0).String(),
				Name:        tool.Name,
				Description: tool.Description,
				Tag:         tool.Tag,
				IsEnabled:   tool.IsEnabled,
			})
		}

		sort.Slice(ret, func(i, j int) bool {
			return ret[i].ToolId < ret[j].ToolId ||
				(ret[i].ToolId == ret[j].ToolId &&
					ret[i].ImportedAt < ret[j].ImportedAt)
		})

		b, err := yaml.Marshal(ret)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to marshal tool results to yaml",
					zap.Error(err))
			panic(err)
		}
		io.WriteString(os.Stdout, string(b[:]))
	},
}
