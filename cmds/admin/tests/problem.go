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
	Use:   "problem",
	Short: "Import and submit a problem",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		if !viper.IsSet("owner-id") {
			panic(fmt.Errorf("owner-id not set"))
		}

		// Step 1: Run import problems command
		fmt.Println("Import problem")
		imports.ProblemsCmd.SetContext(ctx)
		imports.ProblemsCmd.Run(imports.ProblemsCmd, []string{})

		importedProblems := imports.LastImportedProblems
		if len(importedProblems) == 0 {
			fmt.Println("No problems imported")
			return
		}

		// Step 2: Submit and poll each imported problem
		maxRetries := viper.GetInt("max-retries")
		retryInterval := viper.GetDuration("retry-interval")

		var hasFailure bool
		for _, problem := range importedProblems {
			fmt.Printf("\nSubmitting problem: %s (SHA1: %s)\n",
				problem.Name, problem.SHA1)

			// Call submit command
			submit.ProblemCmd.SetContext(ctx)
			submit.ProblemCmd.Run(submit.ProblemCmd, []string{problem.SHA1})

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

			// Fetch problem to get testcase timeouts
			problemOutput := fetchProblem(ctx, lastResult.RefSHA1)

			timeoutMap := make(map[string]int64)
			gracePeriodMap := make(map[string]int64)
			for stageName, stage := range problemOutput.Stages {
				for _, tc := range stage.Testcases {
					key := stageName + "/" + tc.Name
					timeoutMap[key] = tc.Timeout
					gracePeriodMap[key] = tc.GracePeriod
				}
			}

			// Check job results using shared utility
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
				common.AdminShowResultsConfig())

			if hasFailed {
				fmt.Printf("FAILED: Problem %s has failed jobs\n", problem.Name)
				hasFailure = true
			} else {
				fmt.Printf("PASSED: Problem %s (Score: %d/%d)\n",
					problem.Name, lastResult.Score, lastResult.TotalScore)
			}
		}

		if hasFailure {
			os.Exit(1)
		}
	},
}

func init() {
	problemCmd.Flags().
		Int64("owner-id", 0, "Submit as OwnerId")
	// Allow from env
	// problemCmd.MarkFlagRequired("owner-id")

	problemCmd.Flags().
		IntSliceP("problem-id", "i", []int{}, "Problem ids to test")
	problemCmd.MarkFlagRequired("problem-id")

	problemCmd.Flags().
		Int("max-retries", 60, "Max retries for polling results")
	problemCmd.Flags().
		Duration("retry-interval", time.Second, "Interval between retries")

	// Reuse flags from imports.ProblemsCmd
	problemCmd.Flags().AddFlagSet(imports.ProblemsCmd.Flags())
}
