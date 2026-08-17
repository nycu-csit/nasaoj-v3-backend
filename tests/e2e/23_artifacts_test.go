package tests

import (
	"bytes"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

// This suite covers worker-side artifact handling: a testcase can declare
// Artifacts (glob patterns) to keep instead of having its whole working
// directory discarded; kept files are visible to any testcase in a
// dependent (downstream) stage for the rest of the run, but not to
// unrelated, concurrently-running stages; and kept files are retrievable
// afterward via GET /api/artifacts.

var (
	test23SHA1  string
	test23RefId int64
	test23SubId int64
)

func Test_23_1_Import_and_enable_artifacts_problem(t *testing.T) {
	result := ImportAndEnableProblems(t, ProblemsDir(23), TestcasesDir(23))
	if len(result.SHA1s) == 0 {
		t.Fatal("no problems imported")
	}
	test23SHA1 = result.SHA1s[0]
	t.Logf("Imported and enabled problem SHA1: %s", test23SHA1)
}

func Test_23_2_Submit_artifacts_problem(t *testing.T) {
	var submitOutput api.SubmitOutput
	resp, err := client.R().
		SetBody(api.SubmitInput{
			RefSHA1: test23SHA1,
			Type:    models.RefTypeProblem,
		}).
		SetResult(&submitOutput).
		Post("/api/submit")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("failed to submit problem: %v %s", err, resp)
	}

	test23RefId = submitOutput.RefId
	test23SubId = submitOutput.Id
	t.Logf("submitted problem refId: %d, subId: %d", test23RefId, test23SubId)
}

func Test_23_3_Wait_for_completion(t *testing.T) {
	maxRetries := 30
	var result api.ResultOutput

	for range maxRetries {
		resp, err := client.R().
			SetResult(&result).
			SetQueryParam("refId", strconv.FormatInt(test23RefId, 10)).
			SetQueryParam("type", "0"). // RefTypeProblem
			Get("/api/results")
		if err != nil || resp.StatusCode() != http.StatusOK {
			t.Fatalf("failed to get results: %v", resp)
		}

		allDone := true
		for _, job := range result.Jobs {
			switch job.Status {
			case models.StatusPending, models.StatusRunning, models.StatusBlocked:
				allDone = false
			}
		}
		if allDone && len(result.Jobs) > 0 {
			t.Logf("completed. Score: %d/%d", result.Score, result.TotalScore)
			return
		}

		time.Sleep(time.Second)
	}
	t.Fatalf("jobs did not complete within timeout. Last status: %+v", result)
}

func Test_23_4_Child_stage_sees_parent_artifact(t *testing.T) {
	resp, err := client.R().
		SetQueryParam("subId", strconv.FormatInt(test23SubId, 10)).
		SetQueryParam("stage", "child").
		SetQueryParam("testcase", "consume").
		Get("/api/stdout")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("failed to get child stdout: %v %s", err, resp)
	}

	body := string(resp.Body())
	if !strings.Contains(body, "hello from parent") {
		t.Fatalf("expected child (depends on parent) to read parent's kept "+
			"artifact, got: %q", body)
	}
}

func Test_23_5_Sibling_stage_does_not_see_artifact(t *testing.T) {
	resp, err := client.R().
		SetQueryParam("subId", strconv.FormatInt(test23SubId, 10)).
		SetQueryParam("stage", "sibling").
		SetQueryParam("testcase", "unrelated").
		Get("/api/stdout")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("failed to get sibling stdout: %v %s", err, resp)
	}

	body := string(resp.Body())
	if !strings.Contains(body, "ISOLATED") {
		t.Fatalf("expected sibling (unrelated, concurrent stage) to NOT see "+
			"parent's artifact, got: %q", body)
	}
}

// Artifacts are admin-only - the same access level as stderr - since they
// may contain build output/binaries not meant for students to see.

func Test_23_6_Artifact_retrieval_endpoint(t *testing.T) {
	resp, err := adminReq().
		SetQueryParam("subId", strconv.FormatInt(test23SubId, 10)).
		SetQueryParam("stage", "parent").
		SetQueryParam("testcase", "produce").
		Get("/api/admin/artifacts")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("failed to get artifacts: %v %s", err, resp)
	}

	if ct := resp.Header().Get("Content-Type"); ct != "application/x-tar" {
		t.Fatalf("expected Content-Type application/x-tar, got %q", ct)
	}

	body := resp.Body()
	if !bytes.Contains(body, []byte("keep.txt")) {
		t.Fatalf("expected archive to contain keep.txt")
	}
	if bytes.Contains(body, []byte("scratch")) {
		t.Fatalf("expected scratch.txt to be excluded " +
			"(not declared as an artifact)")
	}
}

func Test_23_7_Artifact_retrieval_requires_admin(t *testing.T) {
	// A plain, non-admin user - including the submission's own owner -
	// cannot fetch artifacts at all.
	resp, err := client.R().
		SetQueryParam("subId", strconv.FormatInt(test23SubId, 10)).
		SetQueryParam("stage", "parent").
		SetQueryParam("testcase", "produce").
		Get("/api/admin/artifacts")
	if err != nil {
		t.Fatalf("request errored: %v", err)
	}
	if resp.StatusCode() != http.StatusForbidden {
		t.Fatalf("expected non-admin artifact request to be forbidden, "+
			"got %d %s", resp.StatusCode(), resp.String())
	}

	// There's no non-admin route at all.
	resp, err = client.R().
		SetQueryParam("subId", strconv.FormatInt(test23SubId, 10)).
		SetQueryParam("stage", "parent").
		SetQueryParam("testcase", "produce").
		Get("/api/artifacts")
	if err != nil {
		t.Fatalf("request errored: %v", err)
	}
	if resp.StatusCode() == http.StatusOK {
		t.Fatal("expected /api/artifacts to not exist for non-admins")
	}
}

func Test_23_8_Artifact_retrieval_requires_auth(t *testing.T) {
	anon := resty.New().SetBaseURL(client.BaseURL)
	resp, err := anon.R().
		SetQueryParam("subId", strconv.FormatInt(test23SubId, 10)).
		SetQueryParam("stage", "parent").
		SetQueryParam("testcase", "produce").
		Get("/api/admin/artifacts")
	if err != nil {
		t.Fatalf("request errored: %v", err)
	}
	if resp.StatusCode() == http.StatusOK {
		t.Fatal("expected unauthenticated artifact request to be rejected")
	}
}

func Test_23_9_No_artifacts_kept_returns_404(t *testing.T) {
	resp, err := adminReq().
		SetQueryParam("subId", strconv.FormatInt(test23SubId, 10)).
		SetQueryParam("stage", "child").
		SetQueryParam("testcase", "consume").
		Get("/api/admin/artifacts")
	if err != nil {
		t.Fatalf("request errored: %v", err)
	}
	if resp.StatusCode() != http.StatusNotFound {
		t.Fatalf("expected 404 for a testcase with no declared artifacts, "+
			"got %d", resp.StatusCode())
	}
}

func Test_23_10_Artifact_url_endpoint(t *testing.T) {
	var out struct {
		URL string `json:"url"`
	}
	resp, err := adminReq().
		SetQueryParam("subId", strconv.FormatInt(test23SubId, 10)).
		SetQueryParam("stage", "parent").
		SetQueryParam("testcase", "produce").
		SetResult(&out).
		Get("/api/admin/artifacts/url")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("failed to get artifacts url: %v %s", err, resp)
	}
	if out.URL == "" {
		t.Fatal("expected a non-empty presigned url")
	}

	// A non-admin can't reach the URL-issuing endpoint either.
	resp, err = client.R().
		SetQueryParam("subId", strconv.FormatInt(test23SubId, 10)).
		SetQueryParam("stage", "parent").
		SetQueryParam("testcase", "produce").
		Get("/api/admin/artifacts/url")
	if err != nil {
		t.Fatalf("request errored: %v", err)
	}
	if resp.StatusCode() != http.StatusForbidden {
		t.Fatalf("expected non-admin request to be forbidden, got %d",
			resp.StatusCode())
	}
}
