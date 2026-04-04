package common

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/go-resty/resty/v2"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

const (
	OutputRetryInterval = 500 * time.Millisecond
	DefaultTimeout      = 30 * time.Second
)

// FetchOutput fetches stdout or stderr for a job with retries
// timeout specifies the maximum time to wait for the output
func FetchOutput(
	ctx context.Context,
	endpoint, subId, stage, testcase string,
	timeout time.Duration,
) string {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp := utils.RestyBuilder(ctx, endpoint,
			func(r *resty.Request) *resty.Request {
				return r.
					SetQueryParam("subId", subId).
					SetQueryParam("stage", stage).
					SetQueryParam("testcase", testcase)
			})
		if resp.IsSuccess() {
			return string(resp.Body())
		}
		time.Sleep(OutputRetryInterval)
	}
	return "<no-value>"
}

// ShowResultsConfig configures how results are displayed
type ShowResultsConfig struct {
	StdoutEndpoint string // e.g., "/api/stdout" or "/api/admin/stdout"
	StderrEndpoint string // e.g., "/api/stderr" or "/api/admin/stderr"
}

// JobResult represents a job result for display
type JobResult struct {
	StageName   string
	TestName    string
	Status      models.JobStatus
	Timeout     int64 // testcase timeout in seconds
	GracePeriod int64 // grace period before SIGKILL in seconds
}

// ShowJobResults displays the results of jobs with colored output
// Returns true if any job failed
func ShowJobResults(
	ctx context.Context,
	subId string,
	jobs []JobResult,
	config ShowResultsConfig,
) bool {
	var hasFailed bool
	for _, job := range jobs {
		statusName := models.JobStatusName(job.Status)
		jobColor := color.RGB(0, 200, 0)
		if job.Status != models.StatusSuccess {
			jobColor = color.RGB(255, 165, 0)
			hasFailed = true
		}

		jobColor.Printf("  [%s] %s/%s\n", statusName, job.StageName, job.TestName)

		fetchTimeout := time.Duration(job.Timeout+job.GracePeriod) * time.Second

		// Fetch and print stdout
		stdout := FetchOutput(ctx, config.StdoutEndpoint,
			subId, job.StageName, job.TestName, fetchTimeout)
		if stdout != "" {
			jobColor.Printf("    stdout: |\n%s\n", strings.TrimSpace(stdout))
		}

		// Fetch and print stderr for non-success jobs
		if job.Status != models.StatusSuccess {
			stderr := FetchOutput(ctx, config.StderrEndpoint,
				subId, job.StageName, job.TestName, fetchTimeout)
			if strings.TrimSpace(stderr) != "" {
				jobColor.Printf("    stderr: |\n%s\n", strings.TrimSpace(stderr))
			}
		}
	}
	return hasFailed
}

// ResultSummary contains the summary info for displaying results
type ResultSummary struct {
	Score      int64
	TotalScore int64
}

// ShowResults displays the full result output including summary
// Returns true if any job failed
func ShowResults(
	ctx context.Context,
	subId string,
	summary ResultSummary,
	jobs []JobResult,
	config ShowResultsConfig,
) bool {
	fmt.Printf("Submission: %s (Score: %d/%d)\n",
		subId, summary.Score, summary.TotalScore)
	return ShowJobResults(ctx, subId, jobs, config)
}

// UserShowResultsConfig returns config for user endpoints
func UserShowResultsConfig() ShowResultsConfig {
	return ShowResultsConfig{
		StdoutEndpoint: "/api/stdout",
		StderrEndpoint: "/api/stderr",
	}
}

// AdminShowResultsConfig returns config for admin endpoints
func AdminShowResultsConfig() ShowResultsConfig {
	return ShowResultsConfig{
		StdoutEndpoint: "/api/admin/stdout",
		StderrEndpoint: "/api/admin/stderr",
	}
}
