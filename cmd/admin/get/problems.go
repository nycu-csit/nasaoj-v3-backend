package get

import (
	"io"
	"os"
	"sort"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

type problemResult struct {
	ProblemId  int64  `json:"problemId" yaml:"problemId"`
	SHA1       string `json:"sha1" yaml:"sha1"`
	ImportedAt string `json:"importedAt" yaml:"importedAt"`

	Name        string `json:"name" yaml:"name"`
	Description string `json:"description" yaml:"description"`
	Tag         string `json:"tag" yaml:"tag"`
	IsEnabled   bool   `json:"isEnabled" yaml:"isEnabled"`
}

var problemsCmd = &cobra.Command{
	Use:   "problems",
	Short: "Get problems",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		var res []api.ProblemOutput
		utils.RestyBuilder(ctx, "/api/admin/problems",
			func(r *resty.Request) *resty.Request {
				return r.
					SetResult(&res).
					ForceContentType("application/json")
			})

		var ret []problemResult
		for _, problem := range res {
			ret = append(ret, problemResult{
				ProblemId:   problem.ProblemId,
				SHA1:        problem.SHA1,
				ImportedAt:  time.Unix(problem.ImportedAt, 0).String(),
				Name:        problem.Name,
				Description: problem.Description,
				Tag:         problem.Tag,
				IsEnabled:   problem.IsEnabled,
			})
		}

		sort.Slice(ret, func(i, j int) bool {
			return ret[i].ProblemId < ret[j].ProblemId ||
				(ret[i].ProblemId == ret[j].ProblemId && ret[i].SHA1 < ret[j].SHA1)
		})

		b, err := yaml.Marshal(ret)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to marshal problem results to yaml",
					zap.Error(err))
			panic(err)
		}
		io.WriteString(os.Stdout, string(b[:]))
	},
}
