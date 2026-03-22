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
	problem18SHA1  string
	problem18RefId int64
)

func Test_18_1_Import_container_problem(t *testing.T) {
	cmd := exec.Command(
		"judge", "admin", "import", "problems",
		"--problems-dir", "resources/18/problems",
		"--testcases-dir", "resources/18/testcases",
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

func Test_18_2_Enable_container_problem(t *testing.T) {
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

	const targetProblemId int64 = 19
	found := false
	for _, problem := range problems {
		if problem.ProblemId == targetProblemId {
			found = true
			problem18SHA1 = problem.SHA1
			problem18RefId = problem.ProblemId
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

func Test_18_3_Submit_container_problem(t *testing.T) {
	var submitOutput api.SubmitOutput
	resp, err := client.R().
		SetBody(api.SubmitInput{
			RefSHA1: problem18SHA1,
			Type:    models.RefTypeProblem,
		}).
		SetResult(&submitOutput).
		Post("/api/submit")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Logf("%v", resp)
		t.Fatalf("failed to submit problem: %v", err)
	}

	problem18RefId = submitOutput.RefId
	t.Logf("submitted problem refId: %d, subId: %d", submitOutput.RefId, submitOutput.Id)
}

func Test_18_4_Check_container_results(t *testing.T) {
	maxRetries := 60 // Container jobs may take longer due to image pull
	var result api.ResultOutput

	for range maxRetries {
		resp, err := client.R().
			SetResult(&result).
			SetQueryParam("refId", strconv.FormatInt(problem18RefId, 10)).
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
				SetQueryParam("refId", strconv.FormatInt(problem18RefId, 10)).
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

func Test_18_5_Check_container_stdout(t *testing.T) {
	// Get submission ID
	var subs api.SubmissionsOutput
	resp, err := client.R().
		SetResult(&subs).
		SetQueryParam("refId", strconv.FormatInt(problem18RefId, 10)).
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
