package tests

import (
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"testing"
	"time"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

var (
	test15SHA1      string
	test15RefId     int64
	test15MixedSHA1 string
)

func Test_15_1_Import_and_enable_problem(t *testing.T) {
	result := ImportAndEnableProblems(t, ProblemsDir(15), TestcasesDir(15))
	if len(result.SHA1s) == 0 {
		t.Fatal("no problems imported")
	}
	test15SHA1 = GetProblemSHA1ByID(t, 1501)
	t.Logf("Imported and enabled problem SHA1: %s", test15SHA1)
}

func Test_15_2_Submit_problem(t *testing.T) {
	var submitOutput api.SubmitOutput
	resp, err := client.R().
		SetBody(api.SubmitInput{
			RefSHA1: test15SHA1,
			Type:    models.RefTypeProblem,
		}).
		SetResult(&submitOutput).
		Post("/api/submit")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Logf("%v", resp)
		t.Fatalf("failed to submit problem: %v", err)
	}

	test15RefId = submitOutput.RefId
	t.Logf("submitted problem refId: %d, subId: %d", submitOutput.RefId, submitOutput.Id)
}

func Test_15_3_Check_resource_group_mutex(t *testing.T) {
	// Poll results multiple times and verify that at most 1 job is running at a time
	// within the same resource group
	maxRunningObserved := 0
	maxRetries := 30

	for i := range maxRetries {
		var result api.ResultOutput
		resp, err := client.R().
			SetResult(&result).
			SetQueryParam("refId", strconv.FormatInt(test15RefId, 10)).
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
				SetQueryParam("refId", strconv.FormatInt(test15RefId, 10)).
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

func Test_15_4_Check_timestamps(t *testing.T) {
	// Get submission ID
	var subs api.SubmissionsOutput
	resp, err := client.R().
		SetResult(&subs).
		SetQueryParam("refId", strconv.FormatInt(test15RefId, 10)).
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
		SetQueryParam("problemSHA1", test15SHA1).
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

func Test_15_5_Import_mixed_jobs_problem(t *testing.T) {
	test15MixedSHA1 = GetProblemSHA1ByID(t, 1502)
	t.Logf("Mixed-jobs problem SHA1: %s", test15MixedSHA1)
}

func Test_15_6_Normal_jobs_not_blocked_by_resource_group(t *testing.T) {
	var submitOutput api.SubmitOutput
	resp, err := client.R().
		SetBody(api.SubmitInput{
			RefSHA1: test15MixedSHA1,
			Type:    models.RefTypeProblem,
		}).
		SetResult(&submitOutput).
		Post("/api/submit")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Logf("%v", resp)
		t.Fatalf("failed to submit problem: %v", err)
	}

	refId := submitOutput.RefId
	t.Logf("submitted mixed-jobs problem refId: %d, subId: %d", refId, submitOutput.Id)

	// Verify that normal jobs complete while RG jobs are still running
	normalJobsDone := false
	rgJobsDone := false
	normalDoneWhileRgRunning := false
	maxRetries := 30

	for i := range maxRetries {
		var result api.ResultOutput
		resp, err := client.R().
			SetResult(&result).
			SetQueryParam("refId", strconv.FormatInt(refId, 10)).
			SetQueryParam("type", "0").
			Get("/api/results")
		if err != nil || resp.StatusCode() != http.StatusOK {
			t.Fatalf("failed to get results: %v\n", resp)
		}

		// Check status of normal jobs (normal1, normal2) and resource group jobs (rg1, rg2)
		normalComplete := 0
		rgComplete := 0
		for _, job := range result.Jobs {
			isComplete := job.Status != models.StatusPending &&
				job.Status != models.StatusRunning &&
				job.Status != models.StatusBlocked

			switch job.TestName {
			case "normal1", "normal2":
				if isComplete {
					normalComplete++
				}
			case "rg1", "rg2":
				if isComplete {
					rgComplete++
				}
			}
		}

		t.Logf("poll %d: normal=%d/2, rg=%d/2", i, normalComplete, rgComplete)

		if normalComplete == 2 && !normalJobsDone {
			normalJobsDone = true
			// Check if RG jobs are still running when normal jobs complete
			if rgComplete < 2 {
				normalDoneWhileRgRunning = true
				t.Logf("poll %d: normal jobs completed while RG jobs still running (%d/2)", i, rgComplete)
			}
		}
		if rgComplete == 2 && !rgJobsDone {
			rgJobsDone = true
			t.Logf("poll %d: resource group jobs completed", i)
		}

		if normalJobsDone && rgJobsDone {
			break
		}

		time.Sleep(500 * time.Millisecond)
	}

	if !normalJobsDone || !rgJobsDone {
		t.Fatal("jobs did not complete within timeout")
	}

	if !normalDoneWhileRgRunning {
		t.Fatal("normal jobs should complete while resource group jobs are still running")
	}

	t.Log("Normal jobs completed while resource group jobs were still running")
}
