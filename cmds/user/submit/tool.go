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

func fetchTools(ctx context.Context) []api.ToolOutput {
	var res []api.ToolOutput
	utils.RestyBuilder(ctx, "/api/tools",
		func(r *resty.Request) *resty.Request {
			return r.
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

var toolCmd = &cobra.Command{
	Use:   "tool [tool_sha1]",
	Short: "Submit a tool",
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

		tools := fetchTools(ctx)
		var completions []string
		for _, t := range tools {
			if !t.IsEnabled {
				continue
			}
			desc := fmt.Sprintf("%s\t%s[%d] - %s",
				t.SHA1, t.Name, t.ToolId, t.Description)
			if strings.HasPrefix(t.SHA1, toComplete) ||
				strings.Contains(strings.ToLower(t.Name), strings.ToLower(toComplete)) {
				completions = append(completions, desc)
			}
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		toolSHA1 := args[0]

		// Parse input args
		inputArgs := make(map[string]string)
		for _, arg := range viper.GetStringSlice("arg") {
			strs := strings.Split(arg, "=")
			if len(strs) != 2 {
				fmt.Printf("ERROR: argument format incorrect: %s\n", arg)
				os.Exit(1)
			}
			inputArgs[strs[0]] = strs[1]
		}

		// Step 1: Submit the tool
		fmt.Printf("Submitting tool: %s\n", toolSHA1)

		var submitOutput api.SubmitOutput
		utils.RestyBuilder(ctx, "/api/submit",
			func(r *resty.Request) *resty.Request {
				r.Method = resty.MethodPost
				return r.
					SetBody(api.SubmitInput{
						RefSHA1:   toolSHA1,
						Type:      models.RefTypeTool,
						InputArgs: inputArgs,
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

		// Step 3: Fetch tool to get testcase timeout
		tool := fetchTool(ctx, lastResult.RefSHA1)

		// Step 4: Display results using shared utility
		subIdStr := strconv.FormatInt(submissionId, 10)
		jobs := make([]common.JobResult, len(lastResult.Jobs))
		for i, j := range lastResult.Jobs {
			jobs[i] = common.JobResult{
				StageName:   j.StageName,
				TestName:    j.TestName,
				Status:      j.Status,
				Timeout:     tool.Testcase.Timeout,
				GracePeriod: tool.Testcase.GracePeriod,
			}
		}
		hasFailed := common.ShowJobResults(ctx, subIdStr, jobs,
			common.UserShowResultsConfig())

		if hasFailed {
			fmt.Printf("FAILED: Submission %d has failed jobs\n", submissionId)
			os.Exit(1)
		} else {
			fmt.Printf("PASSED: Submission %d\n", submissionId)
		}
	},
}

func init() {
	toolCmd.Flags().
		StringSliceP("arg", "a", []string{}, "Input args (key=value)")
	toolCmd.Flags().
		Int("max-retries", 60, "Max retries for polling results")
	toolCmd.Flags().
		Duration("retry-interval", time.Second, "Interval between retries")
}
