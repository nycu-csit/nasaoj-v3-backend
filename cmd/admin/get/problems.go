package get

import (
	"fmt"
	"net/url"
	"sort"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	apiHandlers "gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
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

		client, ok := cmd.Context().Value(types.RestyClient("")).(*resty.Client)
		if !ok {
			tracing.Logger.Ctx(ctx).
				Panic("resty client not exist")
			panic(types.ContextNotExistError)
		}

		problemsURL, err := url.JoinPath(
			viper.GetString("endpoint"), "/api/admin/problems")
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to construct url",
					zap.Error(err))
			panic(err)
		}

		var res []apiHandlers.ProblemOutput
		resp, err := client.R().
			SetResult(&res).
			ForceContentType("application/json").
			Get(problemsURL)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to get problems",
					zap.String("url", problemsURL),
					zap.Error(err))
			panic(err)
		}

		if !resp.IsSuccess() {
			tracing.Logger.Ctx(ctx).
				Panic("resp.IsSuccess failed")
			panic(types.HTTPRequestFailedError)
		}

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
		fmt.Printf("%s", string(b[:]))
	},
}
