package tests

import (
	"net/http"
	"strconv"
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
		{1, "80e3f50e9eb66207c6635649ddbeb49bfc0e9f02", 92},
		{2, "499f051d4cf3b54cce7dd3fdb2c56bf81aa57552", 92},
		{3, "9ee0d232bc29ba414babf1be6137ead8b4d5ae19", 0},
		{4, "ca72bba34319ad23376f7486f398d0c0393b012e", 100},
		{5, "6dab14dc1115d372cc3165cb895a3d07842a3a6d", 0},
		{6, "4e0f9603464ffdf1ab9a174d80f0c9f09cf0b454", 0},
		{7, "f91c49464949f044786eeed0f0fd9fabb247561d", 100},
		{8, "8ba191041b56df5da562daf16788547530faea01", 0},
		// {9, "09021c578f7504b1ca209ca4fc03a1a310dd41a8", 100},
		{10, "2e7cadf973e05f53cb7dde7360c87605c2671f0d", 100},
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

			// Get the first stage and first testcase
			var firstStage string
			var firstTestcase string
			for stageName, stage := range problem.Stages {
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

			// Query stdout for the first testcase
			resp, err = client.R().
				SetQueryParam("subId", strconv.FormatInt(subId, 10)).
				SetQueryParam("stage", firstStage).
				SetQueryParam("testcase", firstTestcase).
				Get("/api/stdout")
			if err != nil || resp.StatusCode() != http.StatusOK {
				t.Fatalf("failed to get /api/stdout: %v", resp)
			}

			body := string(resp.Body())
			if len(body) == 0 {
				t.Fatal("stdout is empty")
			}
			t.Logf("stdout: %s", body)
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

			t.Logf("scores for problem %d: %+v", pres.Id, result)
		})
	}
}
