package get

import (
	"io"
	"os"
	"sort"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.uber.org/zap"
	"go.yaml.in/yaml/v3"
)

type problemResultSpec struct {
	Type string `json:"type" yaml:"type"`
	Link string `json:"link" yaml:"link"`
}

type problemResult struct {
	ProblemId  int64  `json:"problemId" yaml:"problemId"`
	SHA1       string `json:"sha1" yaml:"sha1"`
	ImportedAt string `json:"importedAt" yaml:"importedAt"`

	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description" yaml:"description"`
	Tag         string            `json:"tag" yaml:"tag"`
	Spec        problemResultSpec `json:"spec" yaml:"spec"`
	Deadline    string            `json:"deadline" yaml:"deadline"`
	TotalScore  int64             `json:"totalScore" yaml:"totalScore"`
	IsEnabled   bool              `json:"isEnabled" yaml:"isEnabled"`
	ScoreType   int               `json:"scoreType" yaml:"scoreType"`
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

		sort.Slice(res, func(i, j int) bool {
			return res[i].ImportedAt > res[j].ImportedAt
		})
		filter := make(map[int64][]problemResult)
		for _, problem := range res {
			if !viper.GetBool("all") && len(filter[problem.ProblemId]) >= 3 {
				continue
			}
			filter[problem.ProblemId] = append(filter[problem.ProblemId],
				problemResult{
					ProblemId:   problem.ProblemId,
					SHA1:        problem.SHA1,
					ImportedAt:  time.Unix(problem.ImportedAt, 0).String(),
					Name:        problem.Name,
					Description: problem.Description,
					Tag:         problem.Tag,
					Spec: problemResultSpec{
						Type: problem.Spec.Type,
						Link: problem.Spec.Link,
					},
					Deadline:   time.Unix(problem.Deadline, 0).String(),
					TotalScore: problem.TotalScore,
					IsEnabled:  problem.IsEnabled,
					ScoreType:  int(problem.ScoreType),
				})
		}

		var ret []problemResult
		for _, problems := range filter {
			for _, problem := range problems {
				ret = append(ret, problem)
			}
		}

		sort.Slice(ret, func(i, j int) bool {
			return ret[i].ProblemId < ret[j].ProblemId ||
				(ret[i].ProblemId == ret[j].ProblemId &&
					ret[i].ImportedAt < ret[j].ImportedAt)
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

func init() {
	problemsCmd.Flags().
		BoolP("all", "a", false, "Show all problems")
}
