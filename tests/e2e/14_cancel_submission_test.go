package tests

import (
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"go.yaml.in/yaml/v3"
)

var (
	problem14SHA1  string
	problem14RefId int64
)

func Test_14_1_Import_problem(t *testing.T) {
	cmd := exec.Command(
		"judge", "admin", "import", "problems",
		"--problems-dir", "resources/14/problems",
		"--testcases-dir", "resources/14/testcases",
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")

	var (
		stdout strings.Builder
		stderr strings.Builder
	)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Logf("%s\n", stderr.String())
		t.Fatalf("finish with %v\n", err)
	}

	t.Logf("import output: %s", stdout.String())
}

func Test_14_2_Enable_problem(t *testing.T) {
	cmd := exec.Command(
		"judge", "admin", "get", "problems",
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")

	var stdout strings.Builder
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		t.Fatalf("finish with %v\n", err)
	}

	type Problem struct {
		SHA1       string `yaml:"sha1"`
		ProblemId  int64  `yaml:"problemId"`
		ImportedAt string `yaml:"importedAt"`
	}

	var problems []Problem
	if err := yaml.Unmarshal([]byte(stdout.String()), &problems); err != nil {
		t.Fatalf("failed to unmarshal to yaml, %v", err)
	}

	// Find problem with ID 14 (our long problem)
	const targetProblemId int64 = 14
	found := false
	for _, problem := range problems {
		if problem.ProblemId == targetProblemId {
			found = true
			problem14SHA1 = problem.SHA1
			problem14RefId = problem.ProblemId
			t.Logf("Found problem: SHA1=%s, ProblemId=%d", problem.SHA1, problem.ProblemId)

			cmd = exec.Command(
				"judge", "admin", "enable", "problem", problem.SHA1,
				"--access-token", viper.GetString("admin-token"),
				"--endpoint", "http://localhost:8080")

			if err := cmd.Run(); err != nil {
				t.Fatalf("finish with %v\n", err)
			}
			break
		}
	}
	if !found {
		t.Fatalf("problem with ID %d not found in problem list", targetProblemId)
	}
}

func waitForFirstJobComplete(t *testing.T, subId int64) {
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		var result api.ResultOutput
		resp, err := client.R().
			SetResult(&result).
			SetQueryParam("subId", strconv.FormatInt(subId, 10)).
			Get("/api/results")
		if err != nil || resp.StatusCode() != http.StatusOK {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// Check if any job has completed (not Pending, Running, or Blocked)
		for _, job := range result.Jobs {
			switch job.Status {
			case models.StatusSuccess, models.StatusFailed, models.StatusTimeout, models.StatusMissing:
				t.Logf("first job completed with status %d", job.Status)
				return
			}
		}

		time.Sleep(500 * time.Millisecond)
	}
	t.Fatal("timed out waiting for first job to complete")
}

func verifyJobRunningOnWorker(t *testing.T, subId int64) {
	type WorkerStatus struct {
		Name string   `yaml:"name"`
		Jobs []string `yaml:"jobs"`
	}

	jobPrefix := strconv.FormatInt(subId, 10) + "-"

	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		cmd := exec.Command(
			"judge", "admin", "get", "workers",
			"--access-token", viper.GetString("admin-token"),
			"--endpoint", "http://localhost:8080")

		var stdout strings.Builder
		cmd.Stdout = &stdout
		if err := cmd.Run(); err != nil {
			time.Sleep(200 * time.Millisecond)
			continue
		}

		var workers []WorkerStatus
		if err := yaml.Unmarshal([]byte(stdout.String()), &workers); err != nil {
			time.Sleep(200 * time.Millisecond)
			continue
		}

		// Check if any worker has a job from this submission running
		for _, worker := range workers {
			for _, jobId := range worker.Jobs {
				if strings.HasPrefix(jobId, jobPrefix) {
					t.Logf("found job %s running on worker %s", jobId, worker.Name)
					return
				}
			}
		}

		time.Sleep(500 * time.Millisecond)
	}
	t.Fatal("timed out waiting for job to be running on a worker")
}

func Test_14_3_Submit_and_cancel_via_admin_CLI(t *testing.T) {
	// Submit the problem
	var submitOutput api.SubmitOutput
	resp, err := client.R().
		SetBody(api.SubmitInput{
			RefSHA1: problem14SHA1,
			Type:    models.RefTypeProblem,
		}).
		SetResult(&submitOutput).
		Post("/api/submit")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Logf("%v", resp)
		t.Fatalf("failed to submit problem: %v", err)
	}

	subId := submitOutput.Id
	t.Logf("submitted problem refId: %d, subId: %d", submitOutput.RefId, subId)

	// Wait for the first job to complete
	waitForFirstJobComplete(t, subId)

	// Verify the second job is running on a worker
	verifyJobRunningOnWorker(t, subId)

	// Cancel the submission using admin CLI
	cmd := exec.Command(
		"judge", "admin", "cancel", "submission",
		strconv.FormatInt(subId, 10),
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")

	var (
		stdout strings.Builder
		stderr strings.Builder
	)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Logf("stdout: %s", stdout.String())
		t.Logf("stderr: %s", stderr.String())
		t.Fatalf("failed to cancel submission: %v", err)
	}

	t.Logf("cancel output: %s", stdout.String())

	// Wait for cancellation to propagate and second job to return
	var result api.ResultOutput
	maxRetries := 30
	foundCancelledJob := false
	for i := 0; i < maxRetries; i++ {
		resp, err = client.R().
			SetResult(&result).
			SetQueryParam("subId", strconv.FormatInt(subId, 10)).
			Get("/api/results")
		if err != nil || resp.StatusCode() != http.StatusOK {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// Check if we have a failed job (cancelled while running)
		for _, job := range result.Jobs {
			if job.Status == models.StatusFailed && job.RetCode != 0 {
				foundCancelledJob = true
				t.Logf("found cancelled job: stage=%s test=%s retCode=%d",
					job.StageName, job.TestName, job.RetCode)
				break
			}
		}

		if foundCancelledJob {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Count job statuses
	skipCount := 0
	failedCount := 0
	for _, job := range result.Jobs {
		switch job.Status {
		case models.StatusSkipped:
			skipCount++
		case models.StatusFailed:
			failedCount++
		}
	}

	t.Logf("jobs skipped: %d, failed: %d, total: %d", skipCount, failedCount, len(result.Jobs))

	// At least some jobs should be skipped (the pending/blocked ones)
	if skipCount == 0 {
		t.Fatal("expected some jobs to be skipped after cancellation")
	}

	// Should have at least one failed job (the one that was running and got cancelled)
	if !foundCancelledJob {
		t.Fatal("expected a cancelled job with non-zero return code")
	}
}

func Test_14_4_Submit_and_cancel_via_user_API(t *testing.T) {
	// Submit the problem again
	var submitOutput api.SubmitOutput
	resp, err := client.R().
		SetBody(api.SubmitInput{
			RefSHA1: problem14SHA1,
			Type:    models.RefTypeProblem,
		}).
		SetResult(&submitOutput).
		Post("/api/submit")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Logf("%v", resp)
		t.Fatalf("failed to submit problem: %v", err)
	}

	subId := submitOutput.Id
	t.Logf("submitted problem refId: %d, subId: %d", submitOutput.RefId, subId)

	// Wait for the first job to complete
	waitForFirstJobComplete(t, subId)

	// Verify the second job is running on a worker
	verifyJobRunningOnWorker(t, subId)

	// Cancel the submission using user API
	resp, err = client.R().
		SetBody(api.PutCancelSubmissionInput{
			SubId: subId,
		}).
		Put("/api/submission/cancel")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Logf("%v", resp)
		t.Fatalf("failed to cancel submission via API: %v", err)
	}

	t.Logf("cancel API response: %s", string(resp.Body()))

	// Wait for cancellation to propagate and second job to return
	var result api.ResultOutput
	maxRetries := 30
	foundCancelledJob := false
	for i := 0; i < maxRetries; i++ {
		resp, err = client.R().
			SetResult(&result).
			SetQueryParam("subId", strconv.FormatInt(subId, 10)).
			Get("/api/results")
		if err != nil || resp.StatusCode() != http.StatusOK {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// Check if we have a failed job (cancelled while running)
		for _, job := range result.Jobs {
			if job.Status == models.StatusFailed && job.RetCode != 0 {
				foundCancelledJob = true
				t.Logf("found cancelled job: stage=%s test=%s retCode=%d",
					job.StageName, job.TestName, job.RetCode)
				break
			}
		}

		if foundCancelledJob {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Count job statuses
	skipCount := 0
	failedCount := 0
	for _, job := range result.Jobs {
		switch job.Status {
		case models.StatusSkipped:
			skipCount++
		case models.StatusFailed:
			failedCount++
		}
	}

	t.Logf("jobs skipped: %d, failed: %d, total: %d", skipCount, failedCount, len(result.Jobs))

	// At least some jobs should be skipped (the pending/blocked ones)
	if skipCount == 0 {
		t.Fatal("expected some jobs to be skipped after cancellation")
	}

	// Should have at least one failed job (the one that was running and got cancelled)
	if !foundCancelledJob {
		t.Fatal("expected a cancelled job with non-zero return code")
	}
}
