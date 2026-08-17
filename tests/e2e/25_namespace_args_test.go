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
)

// This suite covers namespace-level Args: the outermost layer of the
// problem/stage/testcase args overlay, inherited by every job in the
// namespace unless a more specific level overrides the same key.

const test25Namespace = "ns-args"

var test25SHA1 = map[string]string{} // problem name -> sha1

func Test_25_1_Create_namespace_with_args(t *testing.T) {
	resp, err := adminReq().
		SetBody(map[string]any{
			"name":  test25Namespace,
			"title": "Namespace Args Test",
			"args":  map[string]string{"GREETING": "from-namespace"},
		}).
		Post("/api/admin/namespace")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("failed to create namespace: %v %s", err, resp)
	}
}

func Test_25_2_Import_and_enable_problems(t *testing.T) {
	cmd := exec.Command(
		"judge", "admin", "import", "problems",
		"--problems-dir", ProblemsDir(25),
		"--testcases-dir", TestcasesDir(25),
		"--namespace", test25Namespace,
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to import problems: %v\nstderr: %s", err, stderr.String())
	}

	// Output format: sha1\tproblemId\tname
	for line := range strings.Lines(stdout.String()) {
		fields := strings.SplitN(strings.TrimSpace(line), "\t", 3)
		if len(fields) != 3 {
			continue
		}
		test25SHA1[fields[2]] = fields[0]
	}
	if test25SHA1["Namespace args - no override"] == "" ||
		test25SHA1["Namespace args - override"] == "" {
		t.Fatalf("expected both problems to be imported, got: %+v (raw: %s)",
			test25SHA1, stdout.String())
	}

	for _, sha1 := range test25SHA1 {
		cmd := exec.Command(
			"judge", "admin", "enable", "problem", sha1,
			"--namespace", test25Namespace,
			"--access-token", viper.GetString("admin-token"),
			"--endpoint", "http://localhost:8080")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to enable problem %s: %v", sha1, err)
		}
	}
}

// submitAndGetGreeting submits sha1 into the namespace as a dummy user,
// waits for it to finish, and returns the rendered greeting testcase's
// stdout.
func submitAndGetGreeting(t *testing.T, sha1 string) string {
	t.Helper()

	var submitOut api.SubmitOutput
	resp, err := adminReq().
		SetHeader("X-Namespace", test25Namespace).
		SetBody(map[string]any{
			"refSHA1":   sha1,
			"type":      models.RefTypeProblem,
			"asOwnerId": 1,
		}).
		SetResult(&submitOut).
		Post("/api/admin/submit")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("failed to submit %s: %v %s", sha1, err, resp)
	}

	var result api.ResultOutput
	for range 30 {
		resp, err := adminReq().
			SetHeader("X-Namespace", test25Namespace).
			SetResult(&result).
			SetQueryParam("subId", strconv.FormatInt(submitOut.Id, 10)).
			Get("/api/admin/results")
		if err != nil || resp.StatusCode() != http.StatusOK {
			t.Fatalf("failed to get results: %v %s", err, resp)
		}

		allDone := true
		for _, job := range result.Jobs {
			switch job.Status {
			case models.StatusPending, models.StatusRunning, models.StatusBlocked:
				allDone = false
			}
		}
		if allDone && len(result.Jobs) > 0 {
			break
		}
		time.Sleep(time.Second)
	}

	resp, err = adminReq().
		SetHeader("X-Namespace", test25Namespace).
		SetQueryParam("subId", strconv.FormatInt(submitOut.Id, 10)).
		SetQueryParam("stage", "main").
		SetQueryParam("testcase", "greet").
		Get("/api/admin/stdout")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("failed to get stdout: %v %s", err, resp)
	}
	return resp.String()
}

func Test_25_3_Job_inherits_namespace_arg(t *testing.T) {
	stdout := submitAndGetGreeting(t, test25SHA1["Namespace args - no override"])
	if !strings.Contains(stdout, "greeting: from-namespace") {
		t.Fatalf("expected job to inherit the namespace's GREETING arg, "+
			"got stdout: %q", stdout)
	}
}

func Test_25_4_Problem_arg_overrides_namespace_arg(t *testing.T) {
	stdout := submitAndGetGreeting(t, test25SHA1["Namespace args - override"])
	if !strings.Contains(stdout, "greeting: from-problem") {
		t.Fatalf("expected the problem's own GREETING arg to win over the "+
			"namespace default, got stdout: %q", stdout)
	}
}
