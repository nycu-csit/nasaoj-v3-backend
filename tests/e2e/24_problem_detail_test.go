package tests

import (
	"crypto/sha1"
	"encoding/hex"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"go.yaml.in/yaml/v3"
)

// This suite covers the admin-only "detail" view of a problem: the full
// definition (every field, including testcase file SHA1) as opposed to the
// trimmed summary every other caller gets.

var test24SHA1 string

func Test_24_1_Import_and_enable_detail_problem(t *testing.T) {
	result := ImportAndEnableProblems(t, ProblemsDir(24), TestcasesDir(24))
	if len(result.SHA1s) == 0 {
		t.Fatal("no problems imported")
	}
	test24SHA1 = result.SHA1s[0]
}

func Test_24_2_Summary_omits_detail_fields(t *testing.T) {
	var out api.ProblemOutput
	resp, err := adminReq().
		SetQueryParam("problemSHA1", test24SHA1).
		SetResult(&out).
		Get("/api/admin/problem")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("failed to get problem summary: %v %s", err, resp)
	}

	// The summary shape has no room for cooldown/args/testcase files at all -
	// asserting the raw body doesn't leak them proves detail is opt-in, not
	// just a matter of the struct silently dropping fields on decode.
	body := resp.String()
	if strings.Contains(body, `"cooldown"`) || strings.Contains(body, `"files"`) {
		t.Fatalf("expected summary to omit detail-only fields, got: %s", body)
	}
}

func Test_24_3_Detail_includes_testcase_file_hash(t *testing.T) {
	raw, err := os.ReadFile(TestcasesDir(24) + "/hello.sh")
	if err != nil {
		t.Fatalf("failed to read fixture file: %v", err)
	}
	h := sha1.Sum(raw)
	expectedSHA1 := hex.EncodeToString(h[:])

	var out api.ProblemDetailOutput
	resp, err := adminReq().
		SetQueryParam("problemSHA1", test24SHA1).
		SetQueryParam("detail", "true").
		SetResult(&out).
		Get("/api/admin/problem")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("failed to get problem detail: %v %s", err, resp)
	}

	if out.SHA1 != test24SHA1 {
		t.Fatalf("expected sha1 %s, got %s", test24SHA1, out.SHA1)
	}
	if out.Cooldown != 5 {
		t.Fatalf("expected cooldown 5, got %d", out.Cooldown)
	}
	if out.Args["FOO"] != "bar" {
		t.Fatalf("expected problem arg FOO=bar, got %+v", out.Args)
	}

	stage, ok := out.Stages["main"]
	if !ok || len(stage.Testcases) != 1 {
		t.Fatalf("expected stage 'main' with 1 testcase, got %+v", out.Stages)
	}
	tc := stage.Testcases[0]
	if tc.Args["BAZ"] != "qux" {
		t.Fatalf("expected testcase arg BAZ=qux, got %+v", tc.Args)
	}
	if len(tc.Content.Files) != 1 {
		t.Fatalf("expected 1 content file, got %+v", tc.Content.Files)
	}
	if tc.Content.Files[0].SHA1 != expectedSHA1 {
		t.Fatalf("expected testcase file sha1 %s, got %s",
			expectedSHA1, tc.Content.Files[0].SHA1)
	}
	if tc.Content.Files[0].Name != "hello.sh" {
		t.Fatalf("expected testcase file name hello.sh, got %s",
			tc.Content.Files[0].Name)
	}
}

func Test_24_4_Detail_requires_admin(t *testing.T) {
	// The submission-owning plain user still can't see detail - it's gated
	// on admin, not ownership.
	resp, err := client.R().
		SetQueryParam("problemSHA1", test24SHA1).
		SetQueryParam("detail", "true").
		Get("/api/admin/problem")
	if err != nil {
		t.Fatalf("request errored: %v", err)
	}
	if resp.StatusCode() != http.StatusForbidden {
		t.Fatalf("expected 403 for non-admin, got %d %s",
			resp.StatusCode(), resp.String())
	}

	// The non-admin route ignores an attempted detail=true rather than
	// honoring it - it still only ever returns the summary shape.
	var out api.ProblemOutput
	resp, err = client.R().
		SetQueryParam("problemSHA1", test24SHA1).
		SetQueryParam("detail", "true").
		SetResult(&out).
		Get("/api/problem")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("failed to get problem via non-admin route: %v %s", err, resp)
	}
	if strings.Contains(resp.String(), `"cooldown"`) {
		t.Fatalf("expected non-admin route to ignore detail=true, got: %s",
			resp.String())
	}
}

func Test_24_5_CLI_detail_flag(t *testing.T) {
	cmd := exec.Command(
		"judge", "admin", "get", "problem", test24SHA1, "--detail",
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("judge admin get problem --detail failed: %v\nstderr: %s",
			err, stderr.String())
	}

	var out struct {
		Cooldown int64 `yaml:"cooldown"`
	}
	if err := yaml.Unmarshal([]byte(stdout.String()), &out); err != nil {
		t.Fatalf("failed to unmarshal yaml: %v\noutput: %s", err, stdout.String())
	}
	if out.Cooldown != 5 {
		t.Fatalf("expected cooldown 5 from --detail CLI output, got %d "+
			"(output: %s)", out.Cooldown, stdout.String())
	}
}

func Test_24_6_CLI_problems_detail_flag(t *testing.T) {
	cmd := exec.Command(
		"judge", "admin", "get", "problems", "--detail", "--all",
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("judge admin get problems --detail failed: %v\nstderr: %s",
			err, stderr.String())
	}

	if !strings.Contains(stdout.String(), "cooldown") {
		t.Fatalf("expected --detail problems output to include cooldown, "+
			"got: %s", stdout.String())
	}
}
