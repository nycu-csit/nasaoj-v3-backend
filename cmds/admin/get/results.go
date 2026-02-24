package get

import (
	"io"
	"os"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type resultOutputJob struct {
	StageName  string `yaml:"stageName"`
	TestName   string `yaml:"testName"`
	Status     int64  `yaml:"status"`
	StatusName string `yaml:"statusName"`
	RetCode    int    `yaml:"retCode"`
	ExecAt     int64  `yaml:"execAt"`
}

type resultOutput struct {
	Id         int64             `yaml:"id"`
	RefSHA1    string            `yaml:"refSHA1"`
	RefId      int64             `yaml:"refId"`
	Type       int               `yaml:"type"`
	OwnerId    int64             `yaml:"ownerId"`
	Username   string            `yaml:"username"`
	InputArgs  map[string]string `yaml:"inputArgs"`
	CreatedAt  int64             `yaml:"createdAt"`
	Score      int64             `yaml:"score"`
	TotalScore int64             `yaml:"totalScore"`
	Jobs       []resultOutputJob `yaml:"jobs"`
}

var resultsCmd = &cobra.Command{
	Use:     "results [submission_id]",
	Aliases: []string{"result"},
	Short:   "Get results",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		var apiOutput api.ResultOutput
		utils.RestyBuilder(ctx, "/api/admin/results",
			func(r *resty.Request) *resty.Request {
				return r.
					SetQueryParam("subId", args[0]).
					SetResult(&apiOutput).
					ForceContentType("application/json")
			})

		// Convert to output with status names
		output := resultOutput{
			Id:         apiOutput.Id,
			RefSHA1:    apiOutput.RefSHA1,
			RefId:      apiOutput.RefId,
			Type:       apiOutput.Type,
			OwnerId:    apiOutput.OwnerId,
			Username:   apiOutput.Username,
			InputArgs:  apiOutput.InputArgs,
			CreatedAt:  apiOutput.CreatedAt,
			Score:      apiOutput.Score,
			TotalScore: apiOutput.TotalScore,
			Jobs:       make([]resultOutputJob, len(apiOutput.Jobs)),
		}
		for i, job := range apiOutput.Jobs {
			output.Jobs[i] = resultOutputJob{
				StageName:  job.StageName,
				TestName:   job.TestName,
				Status:     job.Status,
				StatusName: models.JobStatusName(job.Status),
				RetCode:    job.RetCode,
				ExecAt:     job.ExecAt,
			}
		}

		b, err := yaml.Marshal(output)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to marshal output to yaml",
					zap.Error(err))
			panic(err)
		}
		io.WriteString(os.Stdout, string(b[:]))
	},
}
