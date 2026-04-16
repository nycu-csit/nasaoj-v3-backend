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
	test13SHA1  string
	test13RefId int64

	parallelFailSHA1  string
	parallelFailRefId int64
)

// findAndEnableProblemByNameAndId finds a problem by name and ID, enables it, and returns its SHA1
func findAndEnableProblemByNameAndId(t *testing.T, problemId int64, name string) string {
	t.Helper()

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

	var sha1 string
	for _, problem := range problems {
		if problem.ProblemId == problemId && problem.Name == name {
			sha1 = problem.SHA1
			break
		}
	}

	if sha1 == "" {
		t.Fatalf("problem %q (problemId %d) not found", name, problemId)
	}

	t.Logf("found problem %q SHA1: %s", name, sha1)
	EnableProblem(t, sha1)
	return sha1
}

func Test_13_1_Import_and_enable_complex_deps(t *testing.T) {
	ImportProblems(t, ProblemsDir(13), TestcasesDir(13))
	test13SHA1 = findAndEnableProblemByNameAndId(t, 1301, "Complex dependencies")
}

func Test_13_2_Submit_problem(t *testing.T) {
	var submitOutput api.SubmitOutput
	resp, err := client.R().
		SetBody(api.SubmitInput{
			RefSHA1: test13SHA1,
			Type:    models.RefTypeProblem,
		}).
		SetResult(&submitOutput).
		Post("/api/submit")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Logf("%v", resp)
		t.Fatalf("failed to submit problem: %v", err)
	}

	test13RefId = submitOutput.RefId
	t.Logf("submitted problem refId: %d, subId: %d", submitOutput.RefId, submitOutput.Id)
}

func Test_13_3_Check_results(t *testing.T) {
	var result api.ResultOutput
	maxRetries := 30
	for i := range maxRetries {
		resp, err := client.R().
			SetResult(&result).
			SetQueryParam("refId", strconv.FormatInt(test13RefId, 10)).
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
				SetQueryParam("refId", strconv.FormatInt(test13RefId, 10)).
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

func Test_13_4_Enable_parallel_fail_problem(t *testing.T) {
	// Re-import to ensure latest version, then find and enable parallel-fail
	ImportProblems(t, ProblemsDir(13), TestcasesDir(13))
	parallelFailSHA1 = findAndEnableProblemByNameAndId(t, 1302, "Parallel branch failure")
}

func Test_13_5_Submit_parallel_fail_problem(t *testing.T) {
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

func Test_13_6_Check_parallel_fail_results(t *testing.T) {
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
