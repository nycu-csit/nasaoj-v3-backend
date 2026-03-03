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

var (
	problem16SHA1s []string
	problem16RefIds []int64
)

func Test_16_1_Import_problems(t *testing.T) {
	cmd := exec.Command(
		"judge", "admin", "import", "problems",
		"--problems-dir", "resources/16/problems",
		"--testcases-dir", "resources/16/testcases",
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

	for line := range strings.Lines(stdout.String()) {
		line = strings.TrimSpace(line)
		if line != "" {
			problem16SHA1s = append(problem16SHA1s, line)
		}
	}

	if len(problem16SHA1s) != 2 {
		t.Fatalf("expected 2 problems, got %d", len(problem16SHA1s))
	}
	t.Logf("imported problem SHA1s: %v", problem16SHA1s)
}

func Test_16_2_Enable_problems(t *testing.T) {
	for _, sha1 := range problem16SHA1s {
		cmd := exec.Command(
			"judge", "admin", "enable", "problem", sha1,
			"--access-token", viper.GetString("admin-token"),
			"--endpoint", "http://localhost:8080")

		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to enable problem %s: %v", sha1, err)
		}
	}
}

func Test_16_3_Submit_problems(t *testing.T) {
	problem16RefIds = make([]int64, len(problem16SHA1s))

	for i, sha1 := range problem16SHA1s {
		var submitOutput api.SubmitOutput
		resp, err := client.R().
			SetBody(api.SubmitInput{
				RefSHA1: sha1,
				Type:    models.RefTypeProblem,
			}).
			SetResult(&submitOutput).
			Post("/api/submit")
		if err != nil || resp.StatusCode() != http.StatusOK {
			t.Logf("%v", resp)
			t.Fatalf("failed to submit problem %s: %v", sha1, err)
		}
		problem16RefIds[i] = submitOutput.RefId
		t.Logf("submitted problem %s, refId: %d, subId: %d", sha1, submitOutput.RefId, submitOutput.Id)
	}
}

func Test_16_4_Check_results(t *testing.T) {
	for i, refId := range problem16RefIds {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Parallel()

			var result api.ResultOutput
			maxRetries := 30
			for j := range maxRetries {
				resp, err := client.R().
					SetResult(&result).
					SetQueryParam("refId", strconv.FormatInt(refId, 10)).
					SetQueryParam("type", "0"). // RefTypeProblem
					Get("/api/results")
				if err != nil || resp.StatusCode() != http.StatusOK {
					if j < maxRetries-1 {
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
						SetQueryParam("refId", strconv.FormatInt(refId, 10)).
						SetQueryParam("type", "0").
						Get("/api/results")
					if err != nil || resp.StatusCode() != http.StatusOK {
						t.Fatalf("failed to refetch results: %v\n", resp)
					}

					t.Logf("Problem %d completed. Score: %d/%d",
						refId, result.Score, result.TotalScore)

					expectedScore := int64(100)
					if result.Score != expectedScore {
						t.Fatalf("expected score %d, got %d", expectedScore, result.Score)
					}
					return
				}

				time.Sleep(time.Second)
			}

			t.Fatalf("jobs did not complete within timeout. Last status: %+v", result)
		})
	}
}
