package tests

import (
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

// runJudge runs the judge CLI with the given args and returns its stdout.
func runJudge(t *testing.T, args ...string) string {
	t.Helper()

	cmd := exec.Command("judge", args...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("judge %s failed: %v\nstderr: %s",
			strings.Join(args, " "), err, stderr.String())
	}
	return stdout.String()
}

// runJudgeStdin is like runJudge but feeds stdin to the process.
func runJudgeStdin(t *testing.T, stdin string, args ...string) string {
	t.Helper()

	cmd := exec.Command("judge", args...)
	cmd.Stdin = strings.NewReader(stdin)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("judge %s failed: %v\nstderr: %s",
			strings.Join(args, " "), err, stderr.String())
	}
	return stdout.String()
}

// ImportResult holds the result of importing problems
type ImportResult struct {
	SHA1s []string // SHA1s of imported problems in order
}

// ImportProblems imports problems from the given directories and returns the SHA1s
func ImportProblems(t *testing.T, problemsDir, testcasesDir string) ImportResult {
	t.Helper()

	cmd := exec.Command(
		"judge", "admin", "import", "problems",
		"--problems-dir", problemsDir,
		"--testcases-dir", testcasesDir,
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")

	var (
		stdout strings.Builder
		stderr strings.Builder
	)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Logf("stdout: %s\n", stdout.String())
		t.Logf("stderr: %s\n", stderr.String())
		t.Fatalf("failed to import problems: %v", err)
	}

	var sha1s []string
	for line := range strings.Lines(stdout.String()) {
		// Output format: sha1\tproblemId\tname
		fields := strings.Fields(line)
		if len(fields) > 0 {
			sha1s = append(sha1s, fields[0])
		}
	}

	if len(sha1s) == 0 {
		t.Fatal("no problem SHA1s returned from import")
	}

	return ImportResult{SHA1s: sha1s}
}

// EnableProblem enables a problem by its SHA1
func EnableProblem(t *testing.T, sha1 string) {
	t.Helper()

	cmd := exec.Command(
		"judge", "admin", "enable", "problem", sha1,
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")

	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Logf("stderr: %s\n", stderr.String())
		t.Fatalf("failed to enable problem %s: %v", sha1, err)
	}

	t.Logf("Enabled problem SHA1: %s", sha1)
}

// EnableProblems enables multiple problems by their SHA1s
func EnableProblems(t *testing.T, sha1s []string) {
	t.Helper()

	for _, sha1 := range sha1s {
		EnableProblem(t, sha1)
	}
}

// SubmitProblem submits a problem by SHA1 and returns the submission response
func SubmitProblem(t *testing.T, sha1 string) {
	t.Helper()

	resp, err := client.R().
		SetBody(api.SubmitInput{
			RefSHA1: sha1,
			Type:    models.RefTypeProblem,
		}).
		Post("/api/submit")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Logf("response: %v", resp)
		t.Fatalf("failed to submit problem %s: %v", sha1, err)
	}

	t.Logf("Submitted problem SHA1: %s", sha1)
}

// SubmitProblemByID finds a problem by ID and submits it
func SubmitProblemByID(t *testing.T, problemID int64) string {
	t.Helper()

	var problems []api.ProblemOutput
	resp, err := client.R().
		SetResult(&problems).
		Get("/api/problems")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("failed to get problems: %v", resp)
	}

	for _, p := range problems {
		if p.ProblemId == problemID && p.IsEnabled {
			SubmitProblem(t, p.SHA1)
			return p.SHA1
		}
	}

	t.Fatalf("problem %d not found or not enabled", problemID)
	return ""
}

// ImportAndEnableProblems is a convenience function that imports and enables problems
func ImportAndEnableProblems(t *testing.T, problemsDir, testcasesDir string) ImportResult {
	t.Helper()

	result := ImportProblems(t, problemsDir, testcasesDir)
	EnableProblems(t, result.SHA1s)
	return result
}

// GetProblemSHA1ByID returns the SHA1 of an enabled problem by its ID
func GetProblemSHA1ByID(t *testing.T, problemID int64) string {
	t.Helper()

	var problems []api.ProblemOutput
	resp, err := client.R().
		SetResult(&problems).
		Get("/api/problems")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("failed to get problems: %v", resp)
	}

	for _, p := range problems {
		if p.ProblemId == problemID && p.IsEnabled {
			return p.SHA1
		}
	}

	t.Fatalf("problem %d not found or not enabled", problemID)
	return ""
}

// ResourceDir returns the path to the test resource directory for a given test number
func ResourceDir(testNum int) string {
	return fmt.Sprintf("resources/%02d", testNum)
}

// ProblemsDir returns the path to the problems directory for a given test number
func ProblemsDir(testNum int) string {
	return fmt.Sprintf("resources/%02d/problems", testNum)
}

// TestcasesDir returns the path to the testcases directory for a given test number
func TestcasesDir(testNum int) string {
	return fmt.Sprintf("resources/%02d/testcases", testNum)
}
