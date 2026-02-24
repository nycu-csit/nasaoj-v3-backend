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

	// Wait a moment to let the first job start
	time.Sleep(500 * time.Millisecond)

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

	// Wait for the cancellation to propagate
	time.Sleep(time.Second)

	// Check that all jobs are now skipped
	var result api.ResultOutput
	resp, err = client.R().
		SetResult(&result).
		SetQueryParam("refId", strconv.FormatInt(problem14RefId, 10)).
		SetQueryParam("type", "0").
		Get("/api/results")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("failed to get results: %v\n", resp)
	}

	// Count job statuses
	skipCount := 0
	for _, job := range result.Jobs {
		if job.Status == models.StatusSkip {
			skipCount++
		}
	}

	t.Logf("jobs skipped: %d/%d", skipCount, len(result.Jobs))

	// At least some jobs should be skipped (the pending/blocked ones)
	if skipCount == 0 {
		t.Fatal("expected some jobs to be skipped after cancellation")
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

	// Wait a moment to let the first job start
	time.Sleep(500 * time.Millisecond)

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

	// Wait for the cancellation to propagate
	time.Sleep(time.Second)

	// Check that all jobs are now skipped
	var result api.ResultOutput
	resp, err = client.R().
		SetResult(&result).
		SetQueryParam("refId", strconv.FormatInt(problem14RefId, 10)).
		SetQueryParam("type", "0").
		Get("/api/results")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("failed to get results: %v\n", resp)
	}

	// Count job statuses
	skipCount := 0
	for _, job := range result.Jobs {
		if job.Status == models.StatusSkip {
			skipCount++
		}
	}

	t.Logf("jobs skipped: %d/%d", skipCount, len(result.Jobs))

	// At least some jobs should be skipped (the pending/blocked ones)
	if skipCount == 0 {
		t.Fatal("expected some jobs to be skipped after cancellation")
	}
}
