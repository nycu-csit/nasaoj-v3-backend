package tests

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type problemResult struct {
	Id    int64
	SHA1  string
	Score int64
}

var (
	results = []problemResult{
		{201, "feb6c36a5f427d96e589a297ef8704d8867d8ec4", 92},  // hw1
		{202, "3799f9e91d7271b1cf13a13beb2ad186b534be6d", 92},  // hw2
		{203, "3a9ebccf7aac84c510e9a57ac58b7e793d968a91", 0},   // test3
		{204, "b5bbe8a2488c6141013c3339351a6a520ad7c5f4", 100}, // Test stderr
		{205, "a39d2dc0606913f70dc8fbc3d226e67041df43c9", 0},   // Split verdicts
		{206, "3afac455f961371d56ae0ae5819a44fb1dbdafa0", 0},   // Timeout
		{207, "f8c856d773ff99bf69eff129fccdcdde41ec2c4e", 100}, // Go Templates
		{208, "db6269055fa2d1a9e056b96cd636872496799f59", 0},   // Skip jobs
		// {209, "6bf6350c87d69cf51b09e207996c13bf884d2505", 100}, // Long awaiting
		{210, "2658de25c7adc74ee3beca8821104ca86cd3b599", 100}, // pdf
	}
)

func Test_03_1_Check_problems(t *testing.T) {
	var res []api.ProblemOutput
	resp, err := client.R().
		SetResult(&res).
		Get("/api/problems")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("failed to get /api/problems: %v", resp)
	}

	// Check that problemSHA1 is in the list
	for _, pres := range results {
		found := false
		for _, problem := range res {
			if problem.ProblemId == pres.Id {
				if !problem.IsEnabled {
					t.Logf("%v", resp)
					t.Fatalf("problem %d is not enabled", pres.Id)
				}
				if problem.SHA1 != pres.SHA1 {
					t.Logf("%v", resp)
					t.Fatalf("problem %d has mismatched sha1", pres.Id)
				}
				found = true
			}
		}
		if !found {
			t.Fatalf("problem %d not found in /api/problems", pres.Id)
		}
	}
}

func Test_03_2_Run_problems(t *testing.T) {
	for _, pres := range results {
		t.Run(strconv.FormatInt(pres.Id, 10), func(t *testing.T) {
			t.Parallel()
			resp, err := client.R().
				SetBody(api.SubmitInput{
					RefSHA1: pres.SHA1,
					Type:    models.RefTypeProblem,
				}).
				Post("/api/submit")
			if err != nil || resp.StatusCode() != http.StatusOK {
				t.Logf("%v", resp)
				t.Fatalf("failed to submit problem %d: %v", pres.Id, err)
			}
		})
	}
}

func Test_03_3_Check_submissions(t *testing.T) {
	for _, pres := range results {
		t.Run(strconv.FormatInt(pres.Id, 10), func(t *testing.T) {
			t.Parallel()
			var result api.SubmissionsOutput
			resp, err := client.R().
				SetResult(&result).
				SetQueryParam("refId", strconv.FormatInt(pres.Id, 10)).
				SetQueryParam("type", strconv.Itoa(models.RefTypeProblem)).
				Get("/api/submissions")
			if err != nil || resp.StatusCode() != http.StatusOK {
				t.Fatalf("failed to get submissions: %v\n", resp)
			}

			if len(result.Submissions) == 0 {
				t.Fatalf("no submissions found for problem %d", pres.Id)
			}

			// Verify the submission matches our problem
			found := false
			for _, sub := range result.Submissions {
				if sub.RefSHA1 == pres.SHA1 && sub.Type == models.RefTypeProblem {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("submission for problem %d (SHA1: %s) not found", pres.Id, pres.SHA1)
			}
		})
	}
}

func Test_03_4_Check_results(t *testing.T) {
	for _, pres := range results {
		t.Run(strconv.FormatInt(pres.Id, 10), func(t *testing.T) {
			t.Parallel()
			// Poll for results until jobs are done or timeout
			var result api.ResultOutput
			maxRetries := 30
			for i := 0; i < maxRetries; i++ {
				resp, err := client.R().
					SetResult(&result).
					SetQueryParam("refId", strconv.FormatInt(pres.Id, 10)).
					SetQueryParam("type", "0"). // RefTypeProblem
					Get("/api/results")
				if err != nil || resp.StatusCode() != http.StatusOK {
					t.Fatalf("failed to get results: %v\n", resp)
				}

				// Check if all jobs are done
				allDone := true
				for _, job := range result.Jobs {
					switch job.Status {
					case
						models.StatusPending,
						models.StatusRunning,
						models.StatusBlocked:
						allDone = false
					}
				}

				if allDone && len(result.Jobs) > 0 {
					// Wait for cache invalidation
					time.Sleep(2 * time.Second)

					// Refetch to get updated score
					resp, err = client.R().
						SetResult(&result).
						SetQueryParam("refId", strconv.FormatInt(pres.Id, 10)).
						SetQueryParam("type", "0").
						Get("/api/results")
					if err != nil || resp.StatusCode() != http.StatusOK {
						t.Fatalf("failed to refetch results: %v\n", resp)
					}

					t.Logf("Problem %d completed. Score: %d/%d",
						pres.Id, result.Score, result.TotalScore)

					if result.Score != pres.Score {
						t.Fatalf("expected score %d, got %d", pres.Score, result.Score)
					}
					return
				}

				time.Sleep(time.Second)
			}

			t.Fatalf("jobs did not complete within timeout. Last status: %+v", result)
		})
	}
}

func Test_03_5_Check_stdout(t *testing.T) {
	for _, pres := range results {
		t.Run(strconv.FormatInt(pres.Id, 10), func(t *testing.T) {
			t.Parallel()

			// Fetch the problem to get stages and testcases
			var problem api.ProblemOutput
			resp, err := client.R().
				SetResult(&problem).
				SetQueryParam("problemSHA1", pres.SHA1).
				Get("/api/problem")
			if err != nil || resp.StatusCode() != http.StatusOK {
				t.Fatalf("failed to get problem: %v", resp)
			}

			if len(problem.Stages) == 0 {
				t.Fatal("problem has no stages")
			}

			// Get the first stage and first testcase (sorted by stage name)
			stageNames := make([]string, 0, len(problem.Stages))
			for stageName := range problem.Stages {
				stageNames = append(stageNames, stageName)
			}
			sort.Strings(stageNames)

			var firstStage string
			var firstTestcase string
			for _, stageName := range stageNames {
				stage := problem.Stages[stageName]
				if len(stage.Testcases) > 0 {
					firstStage = stageName
					firstTestcase = stage.Testcases[0].Name
					break
				}
			}

			if firstStage == "" || firstTestcase == "" {
				t.Fatal("could not find a stage with testcases")
			}

			// Get submission ID for this problem
			var subs api.SubmissionsOutput
			resp, err = client.R().
				SetResult(&subs).
				SetQueryParam("refId", strconv.FormatInt(pres.Id, 10)).
				SetQueryParam("type", strconv.Itoa(models.RefTypeProblem)).
				Get("/api/submissions")
			if err != nil || resp.StatusCode() != http.StatusOK {
				t.Fatalf("failed to get submissions: %v", resp)
			}

			if len(subs.Submissions) == 0 {
				t.Fatal("no submissions found")
			}

			subId := subs.Submissions[0].Id

			// Query stdout for the first testcase with retry
			var errs strings.Builder
			maxRetries := 10
			for i := 0; i < maxRetries; i++ {
				resp, err = client.R().
					SetQueryParam("subId", strconv.FormatInt(subId, 10)).
					SetQueryParam("stage", firstStage).
					SetQueryParam("testcase", firstTestcase).
					Get("/api/stdout")
				if err == nil && resp.StatusCode() == http.StatusOK {
					body := string(resp.Body())
					if len(body) == 0 {
						t.Fatal("stdout is empty")
					}
					t.Logf("stdout: %s", body)
					return
				}

				errs.WriteString(string(resp.Body()))
				if i < maxRetries-1 {
					time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
				}
			}
			t.Logf("%s", errs.String())
			t.Fatalf("failed to get /api/stdout after %d retries: %v", maxRetries, resp)
		})
	}
}

func Test_03_6_Check_scores(t *testing.T) {
	for _, pres := range results {
		t.Run(strconv.FormatInt(pres.Id, 10), func(t *testing.T) {
			t.Parallel()
			var result []models.UserScore
			resp, err := client.R().
				SetResult(&result).
				SetQueryParam("problemId", strconv.FormatInt(pres.Id, 10)).
				Get("/api/scores")
			if err != nil || resp.StatusCode() != http.StatusOK {
				t.Fatalf("failed to get /api/scores: %v", resp)
			}

			if len(result) == 0 {
				t.Fatal("no scores found")
			}

			// Check that the score matches expected
			if result[0].Score != pres.Score {
				t.Fatalf("expected score %d, got %d", pres.Score, result[0].Score)
			}

			// Check that username is populated
			if result[0].Username == "" {
				t.Fatal("username is empty")
			}

			t.Logf("scores for problem %d: %+v", pres.Id, result)
		})
	}
}
