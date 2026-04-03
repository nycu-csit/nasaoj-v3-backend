package tests

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

var test19SHA1s []string

func Test_19_1_Import_and_enable_timeout_score_problems(t *testing.T) {
	result := ImportAndEnableProblems(t, ProblemsDir(19), TestcasesDir(19))
	test19SHA1s = result.SHA1s
	t.Logf("Imported and enabled %d problems", len(test19SHA1s))
}

func Test_19_2_Submit_timeout_score_problem(t *testing.T) {
	if len(test19SHA1s) == 0 {
		t.Fatal("no problems imported")
	}
	for _, sha1 := range test19SHA1s {
		SubmitProblem(t, sha1)
	}
}

func Test_19_3_Check_timeout_score_not_put(t *testing.T) {
	problemIDs := []int64{20, 21}

	for _, problemID := range problemIDs {
		t.Run("problem_"+string(rune('0'+problemID%10)), func(t *testing.T) {
			var result api.ResultOutput
			maxRetries := 30
			for i := 0; i < maxRetries; i++ {
				resp, err := client.R().
					SetResult(&result).
					SetQueryParam("refId", fmt.Sprintf("%d", problemID)).
					SetQueryParam("type", "0"). // RefTypeProblem
					Get("/api/results")
				if err != nil || resp.StatusCode() != http.StatusOK {
					t.Fatalf("failed to get results: %v\n", resp)
				}

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
						SetQueryParam("refId", fmt.Sprintf("%d", problemID)).
						SetQueryParam("type", "0").
						Get("/api/results")
					if err != nil || resp.StatusCode() != http.StatusOK {
						t.Fatalf("failed to refetch results: %v\n", resp)
					}

					t.Logf("Problem %d completed. Score: %d", problemID, result.Score)

					// Verify job timed out
					for _, job := range result.Jobs {
						if job.Status != models.StatusTimeout {
							t.Fatalf("expected job %s/%s to have status timeout (%d), got %d",
								job.StageName, job.TestName, models.StatusTimeout, job.Status)
						}
					}

					// The score should be 0 because the job timed out before putting the score
					if result.Score != 0 {
						t.Fatalf("expected score 0 (timeout before score put), got %d", result.Score)
					}

					return
				}

				time.Sleep(time.Second)
			}

			t.Fatalf("jobs did not complete within timeout. Last status: %+v", result)
		})
	}
}

func Test_19_4_Check_scores_endpoint_no_score(t *testing.T) {
	problemIDs := []int64{20, 21}

	for _, problemID := range problemIDs {
		t.Run("problem_"+string(rune('0'+problemID%10)), func(t *testing.T) {
			var scores []models.UserScore
			resp, err := client.R().
				SetResult(&scores).
				SetQueryParam("problemId", fmt.Sprintf("%d", problemID)).
				Get("/api/scores")
			if err != nil || resp.StatusCode() != http.StatusOK {
				t.Fatalf("failed to get /api/scores: %v", resp)
			}

			// Should have no scores or score of 0
			if len(scores) > 0 && scores[0].Score != 0 {
				t.Fatalf("expected no score or score 0, got %d", scores[0].Score)
			}

			t.Logf("scores for problem %d: %+v", problemID, scores)
		})
	}
}
