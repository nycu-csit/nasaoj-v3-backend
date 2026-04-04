package tests

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/cmds/admin/imports"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/cmds/admin/submit"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/cmds/common"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

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
	Use:   "tool",
	Short: "Import and submit a tool",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		if !viper.IsSet("owner-id") {
			panic(fmt.Errorf("owner-id not set"))
		}

		// Step 1: Run import tools command
		fmt.Println("Import tool")
		imports.ToolsCmd.SetContext(ctx)
		imports.ToolsCmd.Run(imports.ToolsCmd, []string{})

		importedTools := imports.LastImportedTools
		if len(importedTools) == 0 {
			fmt.Println("No tools imported")
			return
		}

		// Step 2: Submit and poll each imported tool
		maxRetries := viper.GetInt("max-retries")
		retryInterval := viper.GetDuration("retry-interval")

		var hasFailure bool
		for _, tool := range importedTools {
			fmt.Printf("\nSubmitting tool: %s (SHA1: %s)\n", tool.Name, tool.SHA1)

			// Call submit command
			submit.ToolCmd.SetContext(ctx)
			submit.ToolCmd.Run(submit.ToolCmd, []string{tool.SHA1})

			submissionId := submit.LastSubmitOutput.Id
			fmt.Printf("Submission ID: %d\n", submissionId)

			// Step 3: Poll for results
			var lastResult api.ResultOutput
			var allDone bool
			for range maxRetries {
				utils.RestyBuilder(ctx, "/api/admin/results",
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
				hasFailure = true
				continue
			}

			// Fetch tool to get testcase timeout
			toolOutput := fetchTool(ctx, lastResult.RefSHA1)

			// Check job results using shared utility
			subIdStr := strconv.FormatInt(submissionId, 10)
			jobs := make([]common.JobResult, len(lastResult.Jobs))
			for i, j := range lastResult.Jobs {
				jobs[i] = common.JobResult{
					StageName:   j.StageName,
					TestName:    j.TestName,
					Status:      j.Status,
					Timeout:     toolOutput.Testcase.Timeout,
					GracePeriod: toolOutput.Testcase.GracePeriod,
				}
			}
			hasFailed := common.ShowJobResults(ctx, subIdStr, jobs,
				common.AdminShowResultsConfig())

			if hasFailed {
				fmt.Printf("FAILED: Tool %s has failed jobs\n", tool.Name)
				hasFailure = true
			} else {
				fmt.Printf("PASSED: Tool %s (Score: %d/%d)\n",
					tool.Name, lastResult.Score, lastResult.TotalScore)
			}
		}

		if hasFailure {
			os.Exit(1)
		}
	},
}

func init() {
	toolCmd.Flags().
		Int64("owner-id", 0, "Submit as OwnerId")

	toolCmd.Flags().
		IntSliceP("tool-id", "i", []int{}, "Tool ids to test")
	toolCmd.MarkFlagRequired("tool-id")

	toolCmd.Flags().
		Int("max-retries", 60, "Max retries for polling results")
	toolCmd.Flags().
		Duration("retry-interval", time.Second, "Interval between retries")

	// Reuse flags from imports.ToolsCmd
	toolCmd.Flags().AddFlagSet(imports.ToolsCmd.Flags())
	// Reuse flags from submit.ToolCmd
	toolCmd.Flags().AddFlagSet(submit.ToolCmd.Flags())
}
