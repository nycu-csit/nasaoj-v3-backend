package tests

import (
	"net/http"
	"testing"
	"time"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

var test17SHA1s []string

func Test_17_1_Import_and_enable_self_managed_problem(t *testing.T) {
	result := ImportAndEnableProblems(t, ProblemsDir(17), TestcasesDir(17))
	test17SHA1s = result.SHA1s
	t.Logf("Imported and enabled %d problems", len(test17SHA1s))
}

func Test_17_2_Submit_self_managed_problem(t *testing.T) {
	if len(test17SHA1s) == 0 {
		t.Fatal("no problems imported")
	}
	for _, sha1 := range test17SHA1s {
		SubmitProblem(t, sha1)
	}
}

func Test_17_3_Check_self_managed_score(t *testing.T) {
	// Poll for results until jobs are done
	var result api.ResultOutput
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		resp, err := client.R().
			SetResult(&result).
			SetQueryParam("refId", "1701").
			SetQueryParam("type", "0"). // RefTypeProblem
			Get("/api/results")
		if err != nil || resp.StatusCode() != http.StatusOK {
			t.Fatalf("failed to get results: %v\n", resp)
		}

		// Check if all jobs are done
		allDone := true
		for _, job := range result.Jobs {
			switch job.Status {
			case models.StatusPending, models.StatusRunning, models.StatusBlocked:
				allDone = false
			}
		}

		if allDone && len(result.Jobs) > 0 {
			// Wait for cache invalidation
			time.Sleep(2 * time.Second)

			// Refetch to get updated score
			resp, err = client.R().
				SetResult(&result).
				SetQueryParam("refId", "1701").
				SetQueryParam("type", "0").
				Get("/api/results")
			if err != nil || resp.StatusCode() != http.StatusOK {
				t.Fatalf("failed to refetch results: %v\n", resp)
			}

			t.Logf("Problem 1701 completed. Score: %d", result.Score)

			// The testcase script sets score to 42
			expectedScore := int64(42)
			if result.Score != expectedScore {
				t.Fatalf("expected score %d, got %d", expectedScore, result.Score)
			}

			// Verify all jobs succeeded
			for _, job := range result.Jobs {
				if job.Status != models.StatusSuccess {
					t.Fatalf("job %s/%s has unexpected status %d",
						job.StageName, job.TestName, job.Status)
				}
			}

			return
		}

		time.Sleep(time.Second)
	}

	t.Fatalf("jobs did not complete within timeout. Last status: %+v", result)
}

func Test_17_4_Check_scores_endpoint(t *testing.T) {
	var scores []models.UserScore
	resp, err := client.R().
		SetResult(&scores).
		SetQueryParam("problemId", "1701").
		Get("/api/scores")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("failed to get /api/scores: %v", resp)
	}

	if len(scores) == 0 {
		t.Fatal("no scores found")
	}

	// The testcase script sets score to 42
	expectedScore := int64(42)
	if scores[0].Score != expectedScore {
		t.Fatalf("expected score %d, got %d", expectedScore, scores[0].Score)
	}

	t.Logf("scores for problem 1701: %+v", scores)
}
