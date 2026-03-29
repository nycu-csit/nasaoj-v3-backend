package tests

import (
	"net/http"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"go.yaml.in/yaml/v3"
)

func Test_17_1_Import_self_managed_problem(t *testing.T) {
	cmd := exec.Command(
		"judge", "admin", "import", "problems",
		"--problems-dir", "resources/17/problems",
		"--testcases-dir", "resources/17/testcases",
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
		t.Fatalf("failed to import problem: %v\n", err)
	}

	// Output format: sha1\tproblemId\tname
	fields := strings.Fields(stdout.String())
	if len(fields) == 0 {
		t.Fatal("no problem SHA1 returned")
	}
	sha1 := fields[0]
	t.Logf("Imported problem SHA1: %s", sha1)
}

func Test_17_2_Enable_self_managed_problem(t *testing.T) {
	// Get problems list using admin CLI (includes disabled problems)
	cmd := exec.Command(
		"judge", "admin", "get", "problems",
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")

	var stdout strings.Builder
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to get problems: %v", err)
	}

	type Problem struct {
		SHA1      string `yaml:"sha1"`
		ProblemId int64  `yaml:"problemId"`
	}

	var problems []Problem
	if err := yaml.Unmarshal([]byte(stdout.String()), &problems); err != nil {
		t.Fatalf("failed to unmarshal yaml: %v", err)
	}

	// Find problem 18 and enable it
	for _, p := range problems {
		if p.ProblemId == 18 {
			cmd = exec.Command(
				"judge", "admin", "enable", "problem", p.SHA1,
				"--access-token", viper.GetString("admin-token"),
				"--endpoint", "http://localhost:8080")

			if err := cmd.Run(); err != nil {
				t.Fatalf("failed to enable problem: %v", err)
			}
			t.Logf("Enabled problem SHA1: %s", p.SHA1)
			return
		}
	}
	t.Fatal("problem 18 not found")
}

func Test_17_3_Submit_self_managed_problem(t *testing.T) {
	// Get problems to find SHA1
	var problems []api.ProblemOutput
	resp, err := client.R().
		SetResult(&problems).
		Get("/api/problems")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("failed to get problems: %v", resp)
	}

	var problemSHA1 string
	for _, p := range problems {
		if p.ProblemId == 18 && p.IsEnabled {
			problemSHA1 = p.SHA1
			break
		}
	}
	if problemSHA1 == "" {
		t.Fatal("problem 18 not found or not enabled")
	}

	// Submit the problem
	resp, err = client.R().
		SetBody(api.SubmitInput{
			RefSHA1: problemSHA1,
			Type:    models.RefTypeProblem,
		}).
		Post("/api/submit")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Logf("response: %v", resp)
		t.Fatalf("failed to submit problem: %v", err)
	}
	t.Log("Submitted problem 18")
}

func Test_17_4_Check_self_managed_score(t *testing.T) {
	// Poll for results until jobs are done
	var result api.ResultOutput
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		resp, err := client.R().
			SetResult(&result).
			SetQueryParam("refId", "18").
			SetQueryParam("type", "0"). // RefTypeProblem
			Get("/api/results")
		if err != nil || resp.StatusCode() != http.StatusOK {
			t.Fatalf("failed to get results: %v\n", resp)
		}

		// Check if all jobs are done
		allDone := true
		for _, job := range result.Jobs {
			switch job.Status {
			case models.StatusPending, models.StatusRunning, models.StatusBlocked:
				allDone = false
			}
		}

		if allDone && len(result.Jobs) > 0 {
			// Wait for cache invalidation
			time.Sleep(2 * time.Second)

			// Refetch to get updated score
			resp, err = client.R().
				SetResult(&result).
				SetQueryParam("refId", "18").
				SetQueryParam("type", "0").
				Get("/api/results")
			if err != nil || resp.StatusCode() != http.StatusOK {
				t.Fatalf("failed to refetch results: %v\n", resp)
			}

			t.Logf("Problem 18 completed. Score: %d", result.Score)

			// The testcase script sets score to 42
			expectedScore := int64(42)
			if result.Score != expectedScore {
				t.Fatalf("expected score %d, got %d", expectedScore, result.Score)
			}

			// Verify all jobs succeeded
			for _, job := range result.Jobs {
				if job.Status != models.StatusSuccess {
					t.Fatalf("job %s/%s has unexpected status %d",
						job.StageName, job.TestName, job.Status)
				}
			}

			return
		}

		time.Sleep(time.Second)
	}

	t.Fatalf("jobs did not complete within timeout. Last status: %+v", result)
}

func Test_17_5_Check_scores_endpoint(t *testing.T) {
	var scores []models.UserScore
	resp, err := client.R().
		SetResult(&scores).
		SetQueryParam("problemId", "18").
		Get("/api/scores")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("failed to get /api/scores: %v", resp)
	}

	if len(scores) == 0 {
		t.Fatal("no scores found")
	}

	// The testcase script sets score to 42
	expectedScore := int64(42)
	if scores[0].Score != expectedScore {
		t.Fatalf("expected score %d, got %d", expectedScore, scores[0].Score)
	}

	t.Logf("scores for problem 18: %+v", scores)
}
