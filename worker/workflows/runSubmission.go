package workflows

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/worker/cri"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

// Runner holds the shared state for executing claimed submissions. The worker
// now owns per-stage dependency resolution (previously done by the backend).
type Runner struct {
	Root         context.Context
	Client       *resty.Client
	Sem          chan struct{} // bounds concurrent jobs across all submissions
	Secret       []byte        // HMAC key for signing jobTokens (worker token)
	Endpoint     string        // WORKER_API_ENDPOINT advertised to jobs
	SubCancel    *CancelRegistry
	Active       *ActiveSubs
	CriClient    *cri.Client
	WorkerBinary string

	rgMu sync.Map // resourceGroupKey -> *sync.Mutex
}

// ClaimResponse mirrors backend schedule.GetSubmissionOutput.
type ClaimResponse struct {
	Submission models.Submission `json:"submission"`
	Problem    models.Problem    `json:"problem"`
	Tool       models.Tool       `json:"tool"`
}

type marshalJobRequest struct {
	SubId     int64  `json:"subId"`
	StageName string `json:"stageName"`
	TestName  string `json:"testName"`
	JobToken  string `json:"jobToken"`
	Endpoint  string `json:"endpoint"`
}

// ClaimSubmission asks the backend for one submission to process. The bool is
// false when the queue is empty or the request failed.
func ClaimSubmission(
	client *resty.Client,
	pctx context.Context,
) (ClaimResponse, bool) {
	ctx, span := tracing.Tracer.Start(pctx, "worker.ClaimSubmission")
	defer span.End()
	span.SetAttributes(attribute.String("messaging.system", "nasaoj-worker"))

	var claim ClaimResponse
	resp, err := client.R().
		SetContext(ctx).
		SetResult(&claim).
		ForceContentType("application/json").
		SetQueryParam("tags", viper.GetString("tags")).
		Get("/schedule/submission")
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Warn("failed to claim submission",
				zap.Error(err))
		return ClaimResponse{}, false
	}
	if resp.StatusCode() == http.StatusNoContent {
		return ClaimResponse{}, false
	}
	if !resp.IsSuccess() {
		tracing.Logger.Ctx(ctx).
			Warn("claim submission returned non-success",
				zap.Int("status", resp.StatusCode()))
		return ClaimResponse{}, false
	}
	return claim, true
}

func (r *Runner) rgLock(key string) *sync.Mutex {
	m, _ := r.rgMu.LoadOrStore(key, &sync.Mutex{})
	return m.(*sync.Mutex)
}

// RunSubmission executes a whole claimed submission and reports it done.
func (r *Runner) RunSubmission(claim ClaimResponse) {
	sub := claim.Submission

	ctx, span := tracing.Tracer.Start(r.Root, "worker.RunSubmission")
	defer span.End()
	span.SetAttributes(
		attribute.String("messaging.system", "nasaoj-worker"),
		attribute.Int64("submission.id", sub.Id))

	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	subKey := strconv.FormatInt(sub.Id, 10)
	r.SubCancel.Register(subKey, cancel)
	defer r.SubCancel.Unregister(subKey)
	r.Active.Add(sub.Id)
	defer r.Active.Remove(sub.Id)

	// The backend removes the submission from the queue once it observes every
	// job is terminal (in the job-result handler), so there is nothing to
	// report here.
	switch sub.Type {
	case models.RefTypeProblem:
		r.runProblem(subCtx, sub, claim.Problem)
	case models.RefTypeTool:
		r.runOneJob(subCtx, sub, "tool", claim.Tool.Testcase.Name,
			claim.Tool.Testcase.Timeout, 0, "", "")
	}
}

// runProblem resolves the stage DAG: ready stages run; a stage triggers its
// dependents only when all of its testcases succeed; on failure the downstream
// stages are skipped (the backend skips their job rows on the failed result).
func (r *Runner) runProblem(
	ctx context.Context,
	sub models.Submission,
	problem models.Problem,
) {
	indeg := make(map[string]int)
	for _, stage := range problem.Stages {
		for _, next := range stage.Before {
			indeg[next]++
		}
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	skipped := make(map[string]bool)

	var markSkip func(string)
	markSkip = func(name string) {
		for _, next := range problem.Stages[name].Before {
			if !skipped[next] {
				skipped[next] = true
				markSkip(next)
			}
		}
	}

	var runStage func(string)
	runStage = func(name string) {
		defer wg.Done()
		success := r.runStageTestcases(ctx, sub, name, problem.Stages[name])

		mu.Lock()
		defer mu.Unlock()
		if !success {
			markSkip(name)
			return
		}
		for _, next := range problem.Stages[name].Before {
			indeg[next]--
			if indeg[next] == 0 && !skipped[next] {
				wg.Add(1)
				go runStage(next)
			}
		}
	}

	mu.Lock()
	for name := range problem.Stages {
		if indeg[name] == 0 {
			wg.Add(1)
			go runStage(name)
		}
	}
	mu.Unlock()

	wg.Wait()
}

// runStageTestcases runs all testcases of a stage concurrently and reports
// whether the stage fully succeeded. An empty (virtual) stage succeeds.
func (r *Runner) runStageTestcases(
	ctx context.Context,
	sub models.Submission,
	stageName string,
	stage models.Stage,
) bool {
	if len(stage.Testcases) == 0 {
		return true
	}

	results := make([]models.JobStatus, len(stage.Testcases))
	var wg sync.WaitGroup
	for i := range stage.Testcases {
		wg.Add(1)
		go func(i int, tc models.Testcase) {
			defer wg.Done()
			results[i] = r.runOneJob(ctx, sub, stageName, tc.Name,
				tc.Timeout, tc.GracePeriod, tc.ResourceGroup, tc.Image)
		}(i, stage.Testcases[i])
	}
	wg.Wait()

	for _, st := range results {
		if st != models.StatusSuccess {
			return false
		}
	}
	return true
}

// runOneJob marshals, runs, and reports a single job, returning its terminal
// status. Resource-group exclusivity and the global concurrency limit are
// enforced here.
func (r *Runner) runOneJob(
	ctx context.Context,
	sub models.Submission,
	stageName string,
	testName string,
	timeout int64,
	gracePeriod int64,
	resourceGroup string,
	_ string,
) models.JobStatus {
	if resourceGroup != "" {
		lock := r.rgLock(fmt.Sprintf("%d-%s", sub.Id, resourceGroup))
		lock.Lock()
		defer lock.Unlock()
	}

	select {
	case r.Sem <- struct{}{}:
	case <-ctx.Done():
		return models.StatusCancelled
	}
	defer func() { <-r.Sem }()

	if ctx.Err() != nil {
		return models.StatusCancelled
	}

	ttl := time.Duration(timeout+gracePeriod+120) * time.Second
	jobToken, err := SignJobToken(
		r.Secret, sub.Id, stageName, testName, sub.OwnerId, ttl)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to sign job token",
				zap.Error(err))
		return models.StatusFailed
	}

	var job models.Job
	resp, err := r.Client.R().
		SetContext(ctx).
		SetBody(marshalJobRequest{
			SubId:     sub.Id,
			StageName: stageName,
			TestName:  testName,
			JobToken:  jobToken,
			Endpoint:  r.Endpoint,
		}).
		SetResult(&job).
		Post("/schedule/job/marshal")
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to marshal job",
				zap.Error(err))
		return models.StatusFailed
	}
	if resp.StatusCode() == http.StatusConflict {
		// Job already skipped or cancelled by the backend.
		return models.StatusSkipped
	}
	if !resp.IsSuccess() {
		tracing.Logger.Ctx(ctx).
			Error("marshal job returned non-success",
				zap.Int("status", resp.StatusCode()))
		return models.StatusFailed
	}

	tarbuf, err := FetchJobFiles(ctx, &job)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to fetch job files",
				zap.String("job.id", job.Id()),
				zap.Error(err))
		return models.StatusFailed
	}

	jwf := models.JobWithFile{Job: job, Tarbuf: tarbuf}
	retCh := make(chan error, 1)
	go RunJob(ctx, retCh, &jwf, r.CriClient, r.WorkerBinary)
	<-retCh

	r.putResult(jwf.Job)
	return jwf.Job.Status
}

func (r *Runner) putResult(job models.Job) {
	retries := viper.GetInt("putjob-retry")
	for i := 0; i <= retries; i++ {
		resp, err := r.Client.R().
			SetContext(r.Root).
			SetBody(job).
			Put("/schedule/result")
		if err == nil && resp.IsSuccess() {
			return
		}
	}
	tracing.Logger.Ctx(r.Root).
		Error("failed to put job result",
			zap.String("job.id", job.Id()))
}
