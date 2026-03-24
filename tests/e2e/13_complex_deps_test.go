package tests

import (
	"net/http"
	"os/exec"
	"sort"
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
	problem13SHA1  string
	problem13RefId int64

	parallelFailSHA1  string
	parallelFailRefId int64
)

func Test_13_1_Import_problem(t *testing.T) {
	cmd := exec.Command(
		"judge", "admin", "import", "problems",
		"--problems-dir", "resources/13/problems",
		"--testcases-dir", "resources/13/testcases",
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

	t.Logf("imported problem SHA1s:\n%s", stdout.String())
}

func Test_13_2_Enable_problem(t *testing.T) {
	// List all problems to find the one with problemId 12 (Complex dependencies)
	cmd := exec.Command(
		"judge", "admin", "get", "problems",
		"--all",
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")

	var stdout strings.Builder
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to get problems: %v\n", err)
	}

	type Problem struct {
		ProblemId  int64  `yaml:"problemId"`
		SHA1       string `yaml:"sha1"`
		Name       string `yaml:"name"`
		ImportedAt string `yaml:"importedAt"`
	}

	var problems []Problem
	if err := yaml.Unmarshal([]byte(stdout.String()), &problems); err != nil {
		t.Fatalf("failed to unmarshal to yaml: %v", err)
	}

	// Sort by import time descending to get the most recent first
	sort.Slice(problems, func(i, j int) bool {
		return problems[i].ImportedAt > problems[j].ImportedAt
	})

	// Find the most recent problem with problemId 12 (Complex dependencies)
	for _, problem := range problems {
		if problem.ProblemId == 12 && problem.Name == "Complex dependencies" {
			problem13SHA1 = problem.SHA1
			break
		}
	}

	if problem13SHA1 == "" {
		t.Fatal("Complex dependencies problem (problemId 12) not found")
	}

	t.Logf("found Complex dependencies problem SHA1: %s", problem13SHA1)

	// Enable the problem
	cmd = exec.Command(
		"judge", "admin", "enable", "problem", problem13SHA1,
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")

	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Logf("%s\n", stderr.String())
		t.Fatalf("failed to enable problem: %v\n", err)
	}
}

func Test_13_3_Submit_problem(t *testing.T) {
	var submitOutput api.SubmitOutput
	resp, err := client.R().
		SetBody(api.SubmitInput{
			RefSHA1: problem13SHA1,
			Type:    models.RefTypeProblem,
		}).
		SetResult(&submitOutput).
		Post("/api/submit")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Logf("%v", resp)
		t.Fatalf("failed to submit problem: %v", err)
	}

	problem13RefId = submitOutput.RefId
	t.Logf("submitted problem refId: %d, subId: %d", submitOutput.RefId, submitOutput.Id)
}

func Test_13_4_Check_results(t *testing.T) {
	var result api.ResultOutput
	maxRetries := 30
	for i := range maxRetries {
		resp, err := client.R().
			SetResult(&result).
			SetQueryParam("refId", strconv.FormatInt(problem13RefId, 10)).
			SetQueryParam("type", "0"). // RefTypeProblem
			Get("/api/results")
		if err != nil || resp.StatusCode() != http.StatusOK {
			if i < maxRetries-1 {
				time.Sleep(time.Second)
				continue
			}
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
				SetQueryParam("refId", strconv.FormatInt(problem13RefId, 10)).
				SetQueryParam("type", "0").
				Get("/api/results")
			if err != nil || resp.StatusCode() != http.StatusOK {
				t.Fatalf("failed to refetch results: %v\n", resp)
			}

			t.Logf("Problem completed. Score: %d/%d",
				result.Score, result.TotalScore)

			expectedScore := int64(100)
			if result.Score != expectedScore {
				t.Fatalf("expected score %d, got %d", expectedScore, result.Score)
			}
			return
		}

		time.Sleep(time.Second)
	}

	t.Fatalf("jobs did not complete within timeout. Last status: %+v", result)
}

func Test_13_5_Import_parallel_fail_problem(t *testing.T) {
	cmd := exec.Command(
		"judge", "admin", "import", "problems",
		"--problems-dir", "resources/13/problems",
		"--testcases-dir", "resources/13/testcases",
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

	t.Logf("imported SHA1s:\n%s", stdout.String())
}

func Test_13_6_Enable_parallel_fail_problem(t *testing.T) {
	// List all problems to find the one with problemId 13
	cmd := exec.Command(
		"judge", "admin", "get", "problems",
		"--all",
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")

	var stdout strings.Builder
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to get problems: %v\n", err)
	}

	type Problem struct {
		ProblemId  int64  `yaml:"problemId"`
		SHA1       string `yaml:"sha1"`
		Name       string `yaml:"name"`
		IsEnabled  bool   `yaml:"isEnabled"`
		ImportedAt string `yaml:"importedAt"`
	}

	var problems []Problem
	if err := yaml.Unmarshal([]byte(stdout.String()), &problems); err != nil {
		t.Fatalf("failed to unmarshal to yaml: %v", err)
	}

	// Sort by import time descending to get the most recent first
	sort.Slice(problems, func(i, j int) bool {
		return problems[i].ImportedAt > problems[j].ImportedAt
	})

	// Find the most recent problem with problemId 13 (parallel-fail)
	for _, problem := range problems {
		if problem.ProblemId == 13 && problem.Name == "Parallel branch failure" {
			parallelFailSHA1 = problem.SHA1
			break
		}
	}

	if parallelFailSHA1 == "" {
		t.Fatal("parallel-fail problem (problemId 13) not found")
	}

	t.Logf("found parallel-fail problem SHA1: %s", parallelFailSHA1)

	// Enable the problem
	cmd = exec.Command(
		"judge", "admin", "enable", "problem", parallelFailSHA1,
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")

	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Logf("%s\n", stderr.String())
		t.Fatalf("failed to enable problem: %v\n", err)
	}
}

func Test_13_7_Submit_parallel_fail_problem(t *testing.T) {
	var submitOutput api.SubmitOutput
	resp, err := client.R().
		SetBody(api.SubmitInput{
			RefSHA1: parallelFailSHA1,
			Type:    models.RefTypeProblem,
		}).
		SetResult(&submitOutput).
		Post("/api/submit")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Logf("%v", resp)
		t.Fatalf("failed to submit problem: %v", err)
	}

	parallelFailRefId = submitOutput.RefId
	t.Logf("submitted parallel-fail problem refId: %d, subId: %d", submitOutput.RefId, submitOutput.Id)
}

func Test_13_8_Check_parallel_fail_results(t *testing.T) {
	var result api.ResultOutput
	maxRetries := 30
	for i := range maxRetries {
		resp, err := client.R().
			SetResult(&result).
			SetQueryParam("refId", strconv.FormatInt(parallelFailRefId, 10)).
			SetQueryParam("type", "0"). // RefTypeProblem
			Get("/api/results")
		if err != nil || resp.StatusCode() != http.StatusOK {
			if i < maxRetries-1 {
				time.Sleep(time.Second)
				continue
			}
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
				SetQueryParam("refId", strconv.FormatInt(parallelFailRefId, 10)).
				SetQueryParam("type", "0").
				Get("/api/results")
			if err != nil || resp.StatusCode() != http.StatusOK {
				t.Fatalf("failed to refetch results: %v\n", resp)
			}

			t.Logf("Problem completed. Score: %d/%d", result.Score, result.TotalScore)

			// Verify job statuses
			jobStatuses := make(map[string]models.JobStatus)
			for _, job := range result.Jobs {
				key := job.StageName + "/" + job.TestName
				jobStatuses[key] = job.Status
			}

			// Branch A stage 1 should be failed
			if status, ok := jobStatuses["branchA_stage1/fail"]; !ok {
				t.Fatal("branchA_stage1/fail job not found")
			} else if status != models.StatusFailed {
				t.Fatalf("branchA_stage1/fail expected StatusFailed, got %d", status)
			}

			// Branch A stage 2 should be skipped
			if status, ok := jobStatuses["branchA_stage2/should-skip"]; !ok {
				t.Fatal("branchA_stage2/should-skip job not found")
			} else if status != models.StatusSkipped {
				t.Fatalf("branchA_stage2/should-skip expected StatusSkipped, got %d", status)
			}

			// Branch B stage 1 should be success
			if status, ok := jobStatuses["branchB_stage1/succeed"]; !ok {
				t.Fatal("branchB_stage1/succeed job not found")
			} else if status != models.StatusSuccess {
				t.Fatalf("branchB_stage1/succeed expected StatusSuccess, got %d", status)
			}

			// Branch B stage 2 should be success
			if status, ok := jobStatuses["branchB_stage2/succeed"]; !ok {
				t.Fatal("branchB_stage2/succeed job not found")
			} else if status != models.StatusSuccess {
				t.Fatalf("branchB_stage2/succeed expected StatusSuccess, got %d", status)
			}

			// Expected score: branchB_stage1 (25) + branchB_stage2 (25) = 50
			expectedScore := int64(50)
			if result.Score != expectedScore {
				t.Fatalf("expected score %d, got %d", expectedScore, result.Score)
			}
			return
		}

		time.Sleep(time.Second)
	}

	t.Fatalf("jobs did not complete within timeout. Last status: %+v", result)
}
