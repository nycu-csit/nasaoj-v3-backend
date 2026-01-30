package tests

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

var problemId int64

func Test_03_1_Check_problem(t *testing.T) {
	var result []api.ProblemOutput
	resp, err := client.R().
		SetResult(&result).
		Get("/api/problems")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("failed to get /api/problems: %v", resp)
	}

	// Check that problemSHA1 is in the list
	for _, problem := range result {
		if problem.SHA1 == problemSHA1 {
			if !problem.IsEnabled {
				t.Logf("%v", resp)
				t.Fatalf("problem is not enabled")
			}
			problemId = problem.ProblemId
			return
		}
	}

	t.Fatalf("problem with SHA1 %s not found in /api/problems", problemSHA1)
}

func Test_03_2_Run_problem(t *testing.T) {
	resp, err := client.R().
		SetBody(api.SubmitInput{
			RefSHA1: problemSHA1,
			Type:    models.RefTypeProblem,
		}).
		Post("/api/submit")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Logf("%v", resp)
		t.Fatalf("failed to submit, %v\n", err)
	}
}

func Test_03_3_Check_submission(t *testing.T) {
	var result api.SubmissionsOutput
	resp, err := client.R().
		SetResult(&result).
		Get("/api/submissions")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("failed to get submissions: %v\n", resp)
	}

	if len(result.Submissions) == 0 {
		t.Fatal("no submissions found")
	}

	// Verify the submission is for our problem
	found := false
	for _, sub := range result.Submissions {
		if sub.RefSHA1 == problemSHA1 && sub.Type == models.RefTypeProblem {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("submission for problem %s not found", problemSHA1)
	}
}

func Test_03_4_Check_results(t *testing.T) {
	// Poll for results until jobs are done or timeout
	var result api.ResultOutput
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		resp, err := client.R().
			SetResult(&result).
			SetQueryParam("refId", "1").
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
				SetQueryParam("refId", "1").
				SetQueryParam("type", "0").
				Get("/api/results")
			if err != nil || resp.StatusCode() != http.StatusOK {
				t.Fatalf("failed to refetch results: %v\n", resp)
			}

			t.Logf("All jobs completed. Score: %d/%d",
				result.Score, result.TotalScore)
			return
		}

		time.Sleep(time.Second)
	}

	t.Fatalf("jobs did not complete within timeout. Last status: %+v", result)
}

func Test_03_5_Check_stdout(t *testing.T) {
	resp, err := client.R().
		SetQueryParam("subId", "1").
		SetQueryParam("stage", "stage1").
		SetQueryParam("testcase", "test1").
		Get("/api/stdout")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("failed to get /api/stdout: %v", resp)
	}

	// Verify the output contains expected content
	body := string(resp.Body())
	if len(body) == 0 {
		t.Fatal("stdout is empty")
	}
	t.Logf("stdout: %s", body)
}

func Test_03_6_Check_scores(t *testing.T) {
	var result []models.UserScore
	resp, err := client.R().
		SetResult(&result).
		SetQueryParam("problemId", strconv.FormatInt(problemId, 10)).
		Get("/api/scores")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("failed to get /api/scores: %v", resp)
	}

	if len(result) == 0 {
		t.Fatal("no scores found")
	}

	t.Logf("scores: %+v", result)
}
