package user

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/cmds/common"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

func fetchSubmissions(ctx context.Context) []api.SubmissionsOutputSubmission {
	var res api.SubmissionsOutput
	utils.RestyBuilder(ctx, "/api/submissions",
		func(r *resty.Request) *resty.Request {
			return r.
				SetResult(&res).
				ForceContentType("application/json")
		})
	return res.Submissions
}

func fetchProblem(ctx context.Context, sha1 string) api.ProblemOutput {
	var res api.ProblemOutput
	utils.RestyBuilder(ctx, "/api/problem",
		func(r *resty.Request) *resty.Request {
			return r.
				SetQueryParam("problemSHA1", sha1).
				SetResult(&res).
				ForceContentType("application/json")
		})
	return res
}

func fetchTool(ctx context.Context, sha1 string) api.ToolOutput {
	var res api.ToolOutput
	utils.RestyBuilder(ctx, "/api/tool",
		func(r *resty.Request) *resty.Request {
			return r.
				SetQueryParam("toolSHA1", sha1).
				SetResult(&res).
				ForceContentType("application/json")
		})
	return res
}

var resultsCmd = &cobra.Command{
	Use:   "results [submission_id]",
	Short: "Get results for a submission",
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		ctx := cmd.Context()
		utils.ReadConfig(ctx)
		tracing.InitTracer("nasaoj-cli")

		client := resty.New()
		client.SetRedirectPolicy(resty.NoRedirectPolicy())

		// Try to load session for autocomplete with token refresh support
		if session, err := LoadSession(); err == nil && session.RefreshToken != "" {
			SetupClientWithSession(ctx, client, session)
		} else if viper.GetString("access-token") != "" {
			client.SetAuthToken(viper.GetString("access-token"))
		} else {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		ctx = context.WithValue(ctx, types.RestyClient(""), client)

		submissions := fetchSubmissions(ctx)
		slices.Reverse(submissions)
		var completions []string
		for _, s := range submissions {
			idStr := strconv.FormatInt(s.Id, 10)
			createdAt := time.Unix(s.CreatedAt, 0)
			// Format: ID\tmm/dd hh:mm - sha1 - score/totalscore
			desc := fmt.Sprintf("%s\t%s - %s - %d/%d",
				idStr,
				createdAt.Format("01/02 15:04"),
				s.RefSHA1[:8],
				s.Score, s.TotalScore)
			if strings.HasPrefix(idStr, toComplete) {
				completions = append(completions, desc)
			}
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		submissionId := args[0]

		// Fetch results
		var result api.ResultOutput
		utils.RestyBuilder(ctx, "/api/results",
			func(r *resty.Request) *resty.Request {
				return r.
					SetQueryParam("subId", submissionId).
					SetResult(&result).
					ForceContentType("application/json")
			})

		// Build timeout map based on ref type
		timeoutMap := make(map[string]int64)
		switch result.Type {
		case models.RefTypeProblem:
			problem := fetchProblem(ctx, result.RefSHA1)
			for stageName, stage := range problem.Stages {
				for _, tc := range stage.Testcases {
					key := stageName + "/" + tc.Name
					timeoutMap[key] = tc.Timeout
				}
			}
		case models.RefTypeTool:
			tool := fetchTool(ctx, result.RefSHA1)
			// Tools have a single testcase, use its timeout for all jobs
			for _, j := range result.Jobs {
				key := j.StageName + "/" + j.TestName
				timeoutMap[key] = tool.Testcase.Timeout
			}
		}

		// Display results using shared utility
		jobs := make([]common.JobResult, len(result.Jobs))
		for i, j := range result.Jobs {
			key := j.StageName + "/" + j.TestName
			jobs[i] = common.JobResult{
				StageName: j.StageName,
				TestName:  j.TestName,
				Status:    j.Status,
				Timeout:   timeoutMap[key],
			}
		}
		common.ShowResults(ctx, submissionId,
			common.ResultSummary{Score: result.Score, TotalScore: result.TotalScore},
			jobs, common.UserShowResultsConfig())
	},
}

func init() {
	RootCmd.AddCommand(resultsCmd)
}
