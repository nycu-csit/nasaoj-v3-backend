package tests

import (
	"net/http"
	"testing"
	"time"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

// waitForSubmissionComplete waits for all jobs to complete and returns the final score
func waitForSubmissionComplete(t *testing.T) int64 {
	t.Helper()

	var result api.ResultOutput
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		resp, err := client.R().
			SetResult(&result).
			SetQueryParam("refId", "22").
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
			// Wait for cache invalidation
			time.Sleep(2 * time.Second)

			// Refetch to get updated score
			resp, err = client.R().
				SetResult(&result).
				SetQueryParam("refId", "22").
				SetQueryParam("type", "0").
				Get("/api/results")
			if err != nil || resp.StatusCode() != http.StatusOK {
				t.Fatalf("failed to refetch results: %v", resp)
			}

			return result.Score
		}

		time.Sleep(time.Second)
	}

	t.Fatalf("jobs did not complete within timeout")
	return 0
}

// getScoreFromScoreboard returns the score from the scoreboard for problem 22
func getScoreFromScoreboard(t *testing.T) int64 {
	t.Helper()

	var scores []models.UserScore
	resp, err := client.R().
		SetResult(&scores).
		SetQueryParam("problemId", "22").
		Get("/api/scores")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("failed to get /api/scores: %v", resp)
	}

	if len(scores) == 0 {
		t.Fatal("no scores found")
	}

	return scores[0].Score
}

func Test_20_1_Import_v1_score_50(t *testing.T) {
	// Import problem v1 (score=50)
	result := ImportAndEnableProblems(t,
		"resources/20/problems/highest-score-v1.yaml",
		TestcasesDir(20))
	if len(result.SHA1s) == 0 {
		t.Fatal("no problems imported")
	}
	t.Logf("Imported v1 with SHA1: %s", result.SHA1s[0])

	// Submit
	SubmitProblem(t, result.SHA1s[0])

	// Wait for completion
	score := waitForSubmissionComplete(t)
	t.Logf("Submission completed with score: %d", score)

	if score != 50 {
		t.Fatalf("expected score 50, got %d", score)
	}

	// Check scoreboard
	sbScore := getScoreFromScoreboard(t)
	if sbScore != 50 {
		t.Fatalf("expected scoreboard score 50, got %d", sbScore)
	}
	t.Logf("Scoreboard shows score: %d", sbScore)
}

func Test_20_2_Import_v2_score_100_highest_wins(t *testing.T) {
	// Import problem v2 (score=100)
	result := ImportAndEnableProblems(t,
		"resources/20/problems/highest-score-v2.yaml",
		TestcasesDir(20))
	if len(result.SHA1s) == 0 {
		t.Fatal("no problems imported")
	}
	t.Logf("Imported v2 with SHA1: %s", result.SHA1s[0])

	// Submit
	SubmitProblem(t, result.SHA1s[0])

	// Wait for completion
	score := waitForSubmissionComplete(t)
	t.Logf("Submission completed with score: %d", score)

	if score != 100 {
		t.Fatalf("expected score 100, got %d", score)
	}

	// Check scoreboard - should be 100 (highest)
	sbScore := getScoreFromScoreboard(t)
	if sbScore != 100 {
		t.Fatalf("expected scoreboard score 100 (highest), got %d", sbScore)
	}
	t.Logf("Scoreboard shows highest score: %d", sbScore)
}

func Test_20_3_Import_v3_score_25_highest_still_wins(t *testing.T) {
	// Import problem v3 (score=25)
	result := ImportAndEnableProblems(t,
		"resources/20/problems/highest-score-v3.yaml",
		TestcasesDir(20))
	if len(result.SHA1s) == 0 {
		t.Fatal("no problems imported")
	}
	t.Logf("Imported v3 with SHA1: %s", result.SHA1s[0])

	// Submit
	SubmitProblem(t, result.SHA1s[0])

	// Wait for completion
	score := waitForSubmissionComplete(t)
	t.Logf("Submission completed with score: %d", score)

	if score != 25 {
		t.Fatalf("expected score 25, got %d", score)
	}

	// Check scoreboard - should still be 100 (highest from v2)
	sbScore := getScoreFromScoreboard(t)
	if sbScore != 100 {
		t.Fatalf("expected scoreboard score 100 (highest should persist), got %d", sbScore)
	}
	t.Logf("Scoreboard correctly shows highest score: %d (not latest %d)", sbScore, score)
}
