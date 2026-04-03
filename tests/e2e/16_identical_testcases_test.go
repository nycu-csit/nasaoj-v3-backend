package tests

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

var (
	test16SHA1s  []string
	test16RefIds []int64
)

func Test_16_1_Import_and_enable_problems(t *testing.T) {
	result := ImportAndEnableProblems(t, ProblemsDir(16), TestcasesDir(16))
	test16SHA1s = result.SHA1s

	if len(test16SHA1s) != 2 {
		t.Fatalf("expected 2 problems, got %d", len(test16SHA1s))
	}
	t.Logf("imported problem SHA1s: %v", test16SHA1s)
}

func Test_16_2_Submit_problems(t *testing.T) {
	test16RefIds = make([]int64, len(test16SHA1s))

	for i, sha1 := range test16SHA1s {
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
		test16RefIds[i] = submitOutput.RefId
		t.Logf("submitted problem %s, refId: %d, subId: %d", sha1, submitOutput.RefId, submitOutput.Id)
	}
}

func Test_16_3_Check_results(t *testing.T) {
	for i, refId := range test16RefIds {
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
