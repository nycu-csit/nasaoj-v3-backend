package tests

import (
	"net/http"
	"slices"
	"strconv"
	"testing"

	"github.com/go-resty/resty/v2"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/schedule"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

// These tests drive the worker-facing /schedule API directly, acting as a
// fake worker. The problem is tagged "e2e-fake-worker", which the real
// worker (tag "default") never claims, so every submission here is ours.

const (
	test26Tag    = "e2e-fake-worker"
	test26Worker = "e2e-fake-worker-1"
	test26Other  = "e2e-fake-worker-2"
)

var (
	test26SHA1        string
	test26WorkerToken string
)

// scheduleClient returns a client authorized (with a worker token) for the
// /schedule API that identifies itself as the given worker.
func scheduleClient(workerName string) *resty.Client {
	return resty.New().
		SetBaseURL("http://localhost:8080").
		SetAuthToken(test26WorkerToken).
		SetHeader("X-Worker-Name", workerName)
}

func test26Submit(t *testing.T) int64 {
	t.Helper()

	var out api.SubmitOutput
	resp, err := client.R().
		SetBody(api.SubmitInput{
			RefSHA1: test26SHA1,
			Type:    models.RefTypeProblem,
		}).
		SetResult(&out).
		Post("/api/submit")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("failed to submit problem: %v %v", err, resp)
	}
	return out.Id
}

// test26Claim claims the next e2e-fake-worker submission as workerName and
// checks it is the expected one.
func test26Claim(t *testing.T, workerName string, subId int64) {
	t.Helper()

	var out schedule.GetSubmissionOutput
	resp, err := scheduleClient(workerName).R().
		SetQueryParam("tags", test26Tag).
		SetResult(&out).
		Get("/schedule/submission")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("failed to claim submission: %v %v", err, resp)
	}
	if out.Submission.Id != subId {
		t.Fatalf("claimed submission %d, expected %d",
			out.Submission.Id, subId)
	}
}

func test26Marshal(t *testing.T, workerName string, subId int64, stage, test string) {
	t.Helper()

	resp, err := scheduleClient(workerName).R().
		SetBody(schedule.MarshalJobRequest{
			SubId:     subId,
			StageName: stage,
			TestName:  test,
			JobToken:  "unused",
			Endpoint:  "http://127.0.0.1:1",
		}).
		Post("/schedule/job/marshal")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("failed to marshal %s/%s: %v %v", stage, test, err, resp)
	}
}

func test26PutResult(
	t *testing.T, workerName string, subId int64,
	stage, test string, status models.JobStatus,
) *resty.Response {
	t.Helper()

	resp, err := scheduleClient(workerName).R().
		SetBody(models.Job{
			SubId:     subId,
			StageName: stage,
			TestName:  test,
			Status:    status,
		}).
		Put("/schedule/result")
	if err != nil {
		t.Fatalf("failed to put result: %v", err)
	}
	return resp
}

func test26Heartbeat(t *testing.T, workerName string, subIds []int64) schedule.HeartbeatResponse {
	t.Helper()

	var out schedule.HeartbeatResponse
	resp, err := scheduleClient(workerName).R().
		SetBody(schedule.HeartbeatRequest{
			Tags:        []string{test26Tag},
			Submissions: subIds,
		}).
		SetResult(&out).
		Post("/schedule/heartbeat")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("failed to heartbeat: %v %v", err, resp)
	}
	return out
}

func test26Cancel(t *testing.T, subId int64) {
	t.Helper()

	resp, err := client.R().
		SetBody(api.PutCancelSubmissionInput{SubId: subId}).
		Put("/api/submission/cancel")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("failed to cancel submission: %v %v", err, resp)
	}
}

// test26Statuses returns the submission's job statuses keyed by stage name.
func test26Statuses(t *testing.T, subId int64) map[string]models.JobStatus {
	t.Helper()

	var result api.ResultOutput
	resp, err := client.R().
		SetResult(&result).
		SetQueryParam("subId", strconv.FormatInt(subId, 10)).
		Get("/api/results")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("failed to get results: %v %v", err, resp)
	}

	statuses := make(map[string]models.JobStatus)
	for _, job := range result.Jobs {
		statuses[job.StageName] = job.Status
	}
	return statuses
}

func test26Expect(t *testing.T, subId int64, want map[string]models.JobStatus) {
	t.Helper()

	got := test26Statuses(t, subId)
	for stage, status := range want {
		if got[stage] != status {
			t.Fatalf("stage %s: expected status %d, got %d (all: %v)",
				stage, status, got[stage], got)
		}
	}
}

// test26ExpectQueueEmpty asserts no submission is left for the fake worker
// to claim, i.e. the previous one was completed rather than left queued.
func test26ExpectQueueEmpty(t *testing.T) {
	t.Helper()

	resp, err := scheduleClient(test26Worker).R().
		SetQueryParam("tags", test26Tag).
		Get("/schedule/submission")
	if err != nil || resp.StatusCode() != http.StatusNoContent {
		t.Fatalf("expected empty queue, got: %v %v", err, resp)
	}
}

func Test_26_1_Import_and_enable_problem(t *testing.T) {
	test26WorkerToken = genToken(t, "e2e fake worker", "worker")
	result := ImportAndEnableProblems(t, ProblemsDir(26), TestcasesDir(26))
	test26SHA1 = result.SHA1s[0]
}

func Test_26_2_Non_terminal_result_rejected(t *testing.T) {
	subId := test26Submit(t)
	test26Claim(t, test26Worker, subId)
	test26Marshal(t, test26Worker, subId, "first", "one")

	resp := test26PutResult(t, test26Worker, subId,
		"first", "one", models.StatusRunning)
	if resp.StatusCode() != http.StatusBadRequest {
		t.Fatalf("expected 400 for non-terminal result, got %d: %s",
			resp.StatusCode(), resp.String())
	}
	test26Expect(t, subId, map[string]models.JobStatus{
		"first":  models.StatusRunning,
		"second": models.StatusBlocked,
	})

	// A proper terminal result is still accepted afterwards, and finishing
	// both jobs completes the submission.
	resp = test26PutResult(t, test26Worker, subId,
		"first", "one", models.StatusSuccess)
	if resp.StatusCode() != http.StatusOK {
		t.Fatalf("failed to report first: %d %s",
			resp.StatusCode(), resp.String())
	}
	test26Marshal(t, test26Worker, subId, "second", "two")
	resp = test26PutResult(t, test26Worker, subId,
		"second", "two", models.StatusSuccess)
	if resp.StatusCode() != http.StatusOK {
		t.Fatalf("failed to report second: %d %s",
			resp.StatusCode(), resp.String())
	}
	test26Expect(t, subId, map[string]models.JobStatus{
		"first":  models.StatusSuccess,
		"second": models.StatusSuccess,
	})
	test26ExpectQueueEmpty(t)
}

func Test_26_3_Result_from_non_owner_dropped(t *testing.T) {
	subId := test26Submit(t)
	test26Claim(t, test26Worker, subId)
	test26Marshal(t, test26Worker, subId, "first", "one")

	// A late result from a worker that doesn't hold the submission (e.g.
	// one that lost it to a reclaim) is acknowledged but ignored.
	resp := test26PutResult(t, test26Other, subId,
		"first", "one", models.StatusFailed)
	if resp.StatusCode() != http.StatusOK {
		t.Fatalf("expected 200 for non-owner result, got %d: %s",
			resp.StatusCode(), resp.String())
	}
	test26Expect(t, subId, map[string]models.JobStatus{
		"first":  models.StatusRunning,
		"second": models.StatusBlocked,
	})

	// The owner's result is still applied afterwards.
	resp = test26PutResult(t, test26Worker, subId,
		"first", "one", models.StatusFailed)
	if resp.StatusCode() != http.StatusOK {
		t.Fatalf("failed to report owner result: %d %s",
			resp.StatusCode(), resp.String())
	}
	test26Expect(t, subId, map[string]models.JobStatus{
		"first":  models.StatusFailed,
		"second": models.StatusSkipped,
	})
	test26ExpectQueueEmpty(t)
}

func Test_26_4_Blocked_job_failure_completes_submission(t *testing.T) {
	subId := test26Submit(t)
	test26Claim(t, test26Worker, subId)

	// The worker failed before marshal (job still Blocked) and reports it
	// anyway - the result is accepted and the branch below it skipped.
	resp := test26PutResult(t, test26Worker, subId,
		"first", "one", models.StatusFailed)
	if resp.StatusCode() != http.StatusOK {
		t.Fatalf("failed to report blocked job: %d %s",
			resp.StatusCode(), resp.String())
	}
	test26Expect(t, subId, map[string]models.JobStatus{
		"first":  models.StatusFailed,
		"second": models.StatusSkipped,
	})
	test26ExpectQueueEmpty(t)
}

func Test_26_5_Cancel_finishes_orphaned_running_job(t *testing.T) {
	subId := test26Submit(t)

	// Claim and marshal as a worker that has never heartbeated, so no live
	// worker tracks the now-Running job.
	orphanWorker := "e2e-fake-worker-orphan"
	test26Claim(t, orphanWorker, subId)
	test26Marshal(t, orphanWorker, subId, "first", "one")
	test26Expect(t, subId, map[string]models.JobStatus{
		"first": models.StatusRunning,
	})

	// Nobody would ever report it, so cancel must finish it on the spot.
	test26Cancel(t, subId)
	test26Expect(t, subId, map[string]models.JobStatus{
		"first":  models.StatusCancelled,
		"second": models.StatusSkipped,
	})
	test26ExpectQueueEmpty(t)
}

func Test_26_6_Cancel_leaves_tracked_running_job_to_worker(t *testing.T) {
	subId := test26Submit(t)

	// A live (heartbeating) worker tracks the Running job.
	test26Heartbeat(t, test26Worker, nil)
	test26Claim(t, test26Worker, subId)
	test26Marshal(t, test26Worker, subId, "first", "one")

	test26Cancel(t, subId)
	test26Expect(t, subId, map[string]models.JobStatus{
		"first":  models.StatusRunning,
		"second": models.StatusSkipped,
	})

	// The worker learns about the cancel from its heartbeat and reports.
	hb := test26Heartbeat(t, test26Worker, []int64{subId})
	if !slices.Contains(hb.CancelSubmissions, subId) {
		t.Fatalf("expected heartbeat to cancel %d, got %v",
			subId, hb.CancelSubmissions)
	}
	resp := test26PutResult(t, test26Worker, subId,
		"first", "one", models.StatusCancelled)
	if resp.StatusCode() != http.StatusOK {
		t.Fatalf("failed to report cancelled job: %d %s",
			resp.StatusCode(), resp.String())
	}
	test26Expect(t, subId, map[string]models.JobStatus{
		"first":  models.StatusCancelled,
		"second": models.StatusSkipped,
	})
	test26ExpectQueueEmpty(t)
}
