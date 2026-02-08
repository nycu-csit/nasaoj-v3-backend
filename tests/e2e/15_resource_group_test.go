package tests

import (
	"net/http"
	"os/exec"
	"regexp"
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
	problem15SHA1  string
	problem15RefId int64
)

func Test_15_1_Import_problem(t *testing.T) {
	cmd := exec.Command(
		"judge", "admin", "import", "problems",
		"--problems-dir", "resources/15/problems",
		"--testcases-dir", "resources/15/testcases",
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

func Test_15_2_Enable_problem(t *testing.T) {
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

	// Find problem with ID 15 (our resource group test problem)
	const targetProblemId int64 = 15
	found := false
	for _, problem := range problems {
		if problem.ProblemId == targetProblemId {
			found = true
			problem15SHA1 = problem.SHA1
			problem15RefId = problem.ProblemId
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

func Test_15_3_Submit_problem(t *testing.T) {
	var submitOutput api.SubmitOutput
	resp, err := client.R().
		SetBody(api.SubmitInput{
			RefSHA1: problem15SHA1,
			Type:    models.RefTypeProblem,
		}).
		SetResult(&submitOutput).
		Post("/api/submit")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Logf("%v", resp)
		t.Fatalf("failed to submit problem: %v", err)
	}

	problem15RefId = submitOutput.RefId
	t.Logf("submitted problem refId: %d, subId: %d", submitOutput.RefId, submitOutput.Id)
}

func Test_15_4_Check_resource_group_mutex(t *testing.T) {
	// Poll results multiple times and verify that at most 1 job is running at a time
	// within the same resource group
	maxRunningObserved := 0
	maxRetries := 30

	for i := range maxRetries {
		var result api.ResultOutput
		resp, err := client.R().
			SetResult(&result).
			SetQueryParam("refId", strconv.FormatInt(problem15RefId, 10)).
			SetQueryParam("type", "0"). // RefTypeProblem
			Get("/api/results")
		if err != nil || resp.StatusCode() != http.StatusOK {
			t.Fatalf("failed to get results: %v\n", resp)
		}

		// Count running jobs
		runningCount := 0
		for _, job := range result.Jobs {
			if job.Status == models.StatusRunning {
				runningCount++
			}
		}

		if runningCount > maxRunningObserved {
			maxRunningObserved = runningCount
		}

		t.Logf("poll %d: running=%d", i, runningCount)

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
				SetQueryParam("refId", strconv.FormatInt(problem15RefId, 10)).
				SetQueryParam("type", "0").
				Get("/api/results")
			if err != nil || resp.StatusCode() != http.StatusOK {
				t.Fatalf("failed to refetch results: %v\n", resp)
			}

			t.Logf("Problem completed. Score: %d/%d, Max concurrent running: %d",
				result.Score, result.TotalScore, maxRunningObserved)

			// Verify that at most 1 job was running at a time
			if maxRunningObserved > 1 {
				t.Fatalf("expected at most 1 job running at a time, but observed %d", maxRunningObserved)
			}

			// Verify correct score
			if result.Score != 100 {
				t.Fatalf("expected score 100, got %d", result.Score)
			}
			return
		}

		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("jobs did not complete within timeout")
}

func Test_15_5_Check_timestamps(t *testing.T) {
	// Get submission ID
	var subs api.SubmissionsOutput
	resp, err := client.R().
		SetResult(&subs).
		SetQueryParam("refId", strconv.FormatInt(problem15RefId, 10)).
		SetQueryParam("type", strconv.Itoa(models.RefTypeProblem)).
		Get("/api/submissions")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("failed to get submissions: %v", resp)
	}

	if len(subs.Submissions) == 0 {
		t.Fatal("no submissions found")
	}

	subId := subs.Submissions[0].Id

	// Get problem to find testcases
	var problem api.ProblemOutput
	resp, err = client.R().
		SetResult(&problem).
		SetQueryParam("problemSHA1", problem15SHA1).
		Get("/api/problem")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("failed to get problem: %v", resp)
	}

	// Collect timestamps from all testcases
	type timeRange struct {
		testcase string
		start    int64
		end      int64
	}
	var ranges []timeRange

	startRe := regexp.MustCompile(`start (\d+)`)
	endRe := regexp.MustCompile(`end (\d+)`)

	for stageName, stage := range problem.Stages {
		for _, tc := range stage.Testcases {
			resp, err = client.R().
				SetQueryParam("subId", strconv.FormatInt(subId, 10)).
				SetQueryParam("stage", stageName).
				SetQueryParam("testcase", tc.Name).
				Get("/api/stdout")
			if err != nil || resp.StatusCode() != http.StatusOK {
				t.Fatalf("failed to get stdout for %s/%s: %v", stageName, tc.Name, resp)
			}

			body := string(resp.Body())
			t.Logf("stdout for %s/%s: %s", stageName, tc.Name, body)

			startMatch := startRe.FindStringSubmatch(body)
			endMatch := endRe.FindStringSubmatch(body)

			if len(startMatch) < 2 || len(endMatch) < 2 {
				t.Fatalf("could not parse timestamps from stdout: %s", body)
			}

			start, _ := strconv.ParseInt(startMatch[1], 10, 64)
			end, _ := strconv.ParseInt(endMatch[1], 10, 64)

			ranges = append(ranges, timeRange{
				testcase: stageName + "/" + tc.Name,
				start:    start,
				end:      end,
			})
		}
	}

	// Sort by start time
	sort.Slice(ranges, func(i, j int) bool {
		return ranges[i].start < ranges[j].start
	})

	// Verify no overlaps: each job's start should be >= previous job's end
	for i := 1; i < len(ranges); i++ {
		prev := ranges[i-1]
		curr := ranges[i]

		t.Logf("Job %s: start=%d, end=%d", curr.testcase, curr.start, curr.end)

		if curr.start < prev.end {
			t.Fatalf("jobs overlapped: %s (end=%d) and %s (start=%d)",
				prev.testcase, prev.end, curr.testcase, curr.start)
		}
	}

	t.Logf("All %d jobs ran sequentially without overlap", len(ranges))
}
