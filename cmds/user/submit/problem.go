package submit

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/cmds/common"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/cmds/user"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

func fetchProblems(ctx context.Context) []api.ProblemOutput {
	var res []api.ProblemOutput
	utils.RestyBuilder(ctx, "/api/problems",
		func(r *resty.Request) *resty.Request {
			return r.
				SetResult(&res).
				ForceContentType("application/json")
		})
	return res
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

var problemCmd = &cobra.Command{
	Use:   "problem [problem_sha1]",
	Short: "Submit a problem",
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
		if session, err := user.LoadSession(); err == nil && session.RefreshToken != "" {
			user.SetupClientWithSession(ctx, client, session)
		} else if viper.GetString("access-token") != "" {
			client.SetAuthToken(viper.GetString("access-token"))
		} else {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		ctx = context.WithValue(ctx, types.RestyClient(""), client)

		problems := fetchProblems(ctx)
		var completions []string
		for _, p := range problems {
			if !p.IsEnabled {
				continue
			}
			desc := fmt.Sprintf("%s\t%s[%d] - %s",
				p.SHA1, p.Name, p.ProblemId, p.Description)
			if strings.HasPrefix(p.SHA1, toComplete) ||
				strings.Contains(strings.ToLower(p.Name), strings.ToLower(toComplete)) {
				completions = append(completions, desc)
			}
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		problemSHA1 := args[0]

		// Step 1: Submit the problem
		fmt.Printf("Submitting problem: %s\n", problemSHA1)

		var submitOutput api.SubmitOutput
		utils.RestyBuilder(ctx, "/api/submit",
			func(r *resty.Request) *resty.Request {
				r.Method = resty.MethodPost
				return r.
					SetBody(api.SubmitInput{
						RefSHA1: problemSHA1,
						Type:    models.RefTypeProblem,
					}).
					SetResult(&submitOutput).
					ForceContentType("application/json")
			})

		submissionId := submitOutput.Id
		fmt.Printf("Submission ID: %d\n", submissionId)

		// Step 2: Poll for results
		maxRetries := viper.GetInt("max-retries")
		retryInterval := viper.GetDuration("retry-interval")

		var lastResult api.ResultOutput
		var allDone bool
		for range maxRetries {
			utils.RestyBuilder(ctx, "/api/results",
				func(r *resty.Request) *resty.Request {
					return r.
						SetQueryParam("subId", strconv.FormatInt(submissionId, 10)).
						SetResult(&lastResult).
						ForceContentType("application/json")
				})

			allDone = len(lastResult.Jobs) > 0
			for _, job := range lastResult.Jobs {
				if job.Status < models.StatusSuccess {
					allDone = false
					break
				}
			}

			if allDone {
				break
			}

			time.Sleep(retryInterval)
		}

		if !allDone {
			fmt.Printf("TIMEOUT: Jobs did not complete within %d retries\n",
				maxRetries)
			os.Exit(1)
		}

		// Step 3: Fetch problem to get testcase timeouts
		problem := fetchProblem(ctx, lastResult.RefSHA1)

		timeoutMap := make(map[string]int64)
		gracePeriodMap := make(map[string]int64)
		for stageName, stage := range problem.Stages {
			for _, tc := range stage.Testcases {
				key := stageName + "/" + tc.Name
				timeoutMap[key] = tc.Timeout
				gracePeriodMap[key] = tc.GracePeriod
			}
		}

		// Step 4: Display results using shared utility
		subIdStr := strconv.FormatInt(submissionId, 10)
		jobs := make([]common.JobResult, len(lastResult.Jobs))
		for i, j := range lastResult.Jobs {
			key := j.StageName + "/" + j.TestName
			jobs[i] = common.JobResult{
				StageName:   j.StageName,
				TestName:    j.TestName,
				Status:      j.Status,
				Timeout:     timeoutMap[key],
				GracePeriod: gracePeriodMap[key],
			}
		}
		hasFailed := common.ShowJobResults(ctx, subIdStr, jobs,
			common.UserShowResultsConfig())

		if hasFailed {
			fmt.Printf("FAILED: Submission %d has failed jobs\n", submissionId)
			os.Exit(1)
		} else {
			fmt.Printf("PASSED: Submission %d (Score: %d/%d)\n",
				submissionId, lastResult.Score, lastResult.TotalScore)
		}
	},
}

func init() {
	problemCmd.Flags().
		Int("max-retries", 60, "Max retries for polling results")
	problemCmd.Flags().
		Duration("retry-interval", time.Second, "Interval between retries")
}
