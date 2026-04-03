package tests

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

var (
	test18SHA1  string
	test18RefId int64
)

func Test_18_1_Import_and_enable_container_problem(t *testing.T) {
	result := ImportAndEnableProblems(t, ProblemsDir(18), TestcasesDir(18))
	if len(result.SHA1s) == 0 {
		t.Fatal("no problems imported")
	}
	test18SHA1 = result.SHA1s[0]
	t.Logf("Imported and enabled problem SHA1: %s", test18SHA1)
}

func Test_18_2_Submit_container_problem(t *testing.T) {
	var submitOutput api.SubmitOutput
	resp, err := client.R().
		SetBody(api.SubmitInput{
			RefSHA1: test18SHA1,
			Type:    models.RefTypeProblem,
		}).
		SetResult(&submitOutput).
		Post("/api/submit")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Logf("%v", resp)
		t.Fatalf("failed to submit problem: %v", err)
	}

	test18RefId = submitOutput.RefId
	t.Logf("submitted problem refId: %d, subId: %d", submitOutput.RefId, submitOutput.Id)
}

func Test_18_3_Check_container_results(t *testing.T) {
	maxRetries := 60 // Container jobs may take longer due to image pull
	var result api.ResultOutput

	for range maxRetries {
		resp, err := client.R().
			SetResult(&result).
			SetQueryParam("refId", strconv.FormatInt(test18RefId, 10)).
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
				SetQueryParam("refId", strconv.FormatInt(test18RefId, 10)).
				SetQueryParam("type", "0").
				Get("/api/results")
			if err != nil || resp.StatusCode() != http.StatusOK {
				t.Fatalf("failed to refetch results: %v\n", resp)
			}

			t.Logf("Container problem completed. Score: %d/%d",
				result.Score, result.TotalScore)

			// Verify correct score
			if result.Score != 100 {
				t.Fatalf("expected score 100, got %d", result.Score)
			}
			return
		}

		time.Sleep(time.Second)
	}

	t.Fatalf("jobs did not complete within timeout. Last status: %+v", result)
}

func Test_18_4_Check_container_stdout(t *testing.T) {
	// Get submission ID
	var subs api.SubmissionsOutput
	resp, err := client.R().
		SetResult(&subs).
		SetQueryParam("refId", strconv.FormatInt(test18RefId, 10)).
		SetQueryParam("type", strconv.Itoa(models.RefTypeProblem)).
		Get("/api/submissions")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("failed to get submissions: %v", resp)
	}

	if len(subs.Submissions) == 0 {
		t.Fatal("no submissions found")
	}

	subId := subs.Submissions[0].Id

	// Query stdout for the container testcase with retry
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		resp, err = client.R().
			SetQueryParam("subId", strconv.FormatInt(subId, 10)).
			SetQueryParam("stage", "container").
			SetQueryParam("testcase", "alpine-test").
			Get("/api/stdout")
		if err == nil && resp.StatusCode() == http.StatusOK {
			body := string(resp.Body())
			if len(body) == 0 {
				t.Fatal("stdout is empty")
			}

			t.Logf("stdout: %s", body)

			// Verify container executed correctly
			if !strings.Contains(body, "Running inside container") {
				t.Fatalf("expected 'Running inside container' in stdout, got: %s", body)
			}
			if !strings.Contains(body, "Success!") {
				t.Fatalf("expected 'Success!' in stdout, got: %s", body)
			}
			return
		}

		if i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
		}
	}
	t.Fatalf("failed to get /api/stdout after %d retries: %v", maxRetries, resp)
}
