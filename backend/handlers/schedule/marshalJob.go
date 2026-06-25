package schedule

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// MarshalJobRequest is the information a worker supplies to have the backend
// assemble (marshal) a single job: identity plus the worker-signed jobToken and
// the worker API endpoint to bake into the job tar.
type MarshalJobRequest struct {
	SubId     int64  `json:"subId"`
	StageName string `json:"stageName"`
	TestName  string `json:"testName"`
	JobToken  string `json:"jobToken"`
	Endpoint  string `json:"endpoint"`
}

// PostScheduleMarshalJob
//
//	@Summary		Marshal a single job
//	@Description	Builds the ref/job tars for one job on demand from the
//	@Description	worker-supplied identity, marks the job Running, and returns
//	@Description	the job with presigned download URLs.
//	@Accept			json
//	@Produce		json
//	@Param			X-Worker-Name	header		string				true	"worker name"
//	@Param			request			body		MarshalJobRequest	true	"job identity, jobToken, endpoint"
//	@Success		200				{object}	models.Job			"marshalled job"
//	@Failure		400				{object}	string				"bad request"
//	@Failure		409				{object}	string				"job already skipped or cancelled"
//	@Failure		503				{object}	string				"job update failed"
//	@Router			/schedule/job/marshal [post]
func (self *Handlers) PostScheduleMarshalJob(
	w http.ResponseWriter, req bunrouter.Request,
) error {
	ctx := req.Context()
	workerName := req.Header.Get("X-Worker-Name")

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "data retrieve failed",
			OriginError: err,
		}
	}

	var in MarshalJobRequest
	if err := json.Unmarshal(body, &in); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Message:     "json unmarshal failed",
			OriginError: err,
		}
	}

	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.Int64("submission.id", in.SubId),
		attribute.String("stage.name", in.StageName),
		attribute.String("testcase.name", in.TestName))

	submission, err := self.submissions.GetSubmission(ctx, in.SubId)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "failed to get submission",
			OriginError: err,
		}
	}

	var (
		contents []models.Content
		job      models.Job
		args     map[string]string
		input    string
	)

	injected := map[string]string{
		"subId":     strconv.FormatInt(submission.Id, 10),
		"jobToken":  in.JobToken,
		"endpoint":  in.Endpoint,
		"jobInput":  "",
		"jobScript": "",
	}

	switch submission.Type {
	case models.RefTypeProblem:
		problem, err := self.problems.GetProblem(ctx, submission.ProblemSHA1)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     "problem not exist",
				OriginError: err,
			}
		}
		if err := self.contents.HydrateProblem(ctx, &problem); err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "failed to hydrate problem",
				OriginError: err,
			}
		}

		stage, ok := problem.Stages[in.StageName]
		if !ok {
			return middlewares.HTTPError{
				StatusCode: http.StatusBadRequest,
				Message:    "stage not exist",
			}
		}

		var testcase *models.Testcase
		for i := range stage.Testcases {
			if stage.Testcases[i].Name == in.TestName {
				testcase = &stage.Testcases[i]
				break
			}
		}
		if testcase == nil {
			return middlewares.HTTPError{
				StatusCode: http.StatusBadRequest,
				Message:    "testcase not exist",
			}
		}

		contents = []models.Content{
			problem.PreScript,
			stage.PreScript,
			testcase.Content,
			stage.PostScript,
			problem.PostScript,
		}
		job = models.Job{
			SubId:     submission.Id,
			StageName: in.StageName,
			TestName:  testcase.Name,

			Timeout:       testcase.Timeout,
			GracePeriod:   testcase.GracePeriod,
			RefSHA1:       submission.ProblemSHA1,
			ResourceGroup: testcase.ResourceGroup,
			Image:         testcase.Image,
		}
		injected["jobInput"] = job.Id() + ".input"
		injected["jobScript"] = job.Id() + ".bash"
		args = utils.ArgsOverlay(
			submission.Args,
			problem.Args,
			stage.Args,
			testcase.Args,
			injected,
		)

	case models.RefTypeTool:
		tool, err := self.tools.GetTool(ctx, submission.ToolSHA1)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     "tool not exist",
				OriginError: err,
			}
		}
		if err := self.contents.HydrateTool(ctx, &tool); err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "failed to hydrate tool",
				OriginError: err,
			}
		}

		b, err := json.Marshal(submission.InputArgs)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "failed to marshal input args",
				OriginError: err,
			}
		}
		input = string(b)

		contents = []models.Content{
			tool.PreScript,
			tool.Testcase.Content,
			tool.PostScript,
		}
		job = models.Job{
			SubId:     submission.Id,
			StageName: "tool",
			TestName:  tool.Testcase.Name,

			Input:   input,
			Timeout: tool.Testcase.Timeout,
			RefSHA1: submission.ToolSHA1,
		}
		injected["jobInput"] = job.Id() + ".input"
		injected["jobScript"] = job.Id() + ".bash"
		args = utils.ArgsOverlay(
			submission.Args,
			tool.Args,
			tool.Testcase.Args,
			injected,
		)
	default:
		return middlewares.HTTPError{
			StatusCode: http.StatusBadRequest,
			Message:    "unknown submission type",
		}
	}

	jobScript := ""
	for _, content := range contents {
		for _, script := range content.Scripts {
			jobScript += script + "\n"
		}
	}
	job.Script = jobScript

	// Mark the job running (Blocked -> Running). Refuse if already skipped or
	// cancelled so the worker knows to drop it.
	job.Status = models.StatusRunning
	if err := self.jobs.UpdateJob(ctx, &job, models.StatusBlocked); err != nil {
		if errors.Is(err, types.JobAlreadySkippedError) {
			return middlewares.HTTPError{
				StatusCode: http.StatusConflict,
				Message:    "job already skipped or cancelled",
			}
		}
		return middlewares.HTTPError{
			StatusCode:  http.StatusServiceUnavailable,
			Message:     "job update failed",
			OriginError: err,
		}
	}

	// Track the job on the worker so cancellation / orphan detection works.
	if err := self.workers.AddWorkerJob(ctx, workerName, job.Id()); err != nil {
		tracing.Logger.Ctx(ctx).
			Warn("failed to add worker job",
				zap.String("worker", workerName),
				zap.String("job.id", job.Id()),
				zap.Error(err))
	}

	// Write the ref tar (problem/tool content) once per ref sha.
	exist, err := self.s3.ExistFile(ctx, job.RefFilename())
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "failed to check ref file",
			OriginError: err,
		}
	}
	if !exist {
		var refbuf bytes.Buffer
		if err := utils.WriteRefTar(ctx, &refbuf, contents); err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "failed to write ref tar",
				OriginError: err,
			}
		}
		if err := self.s3.PutFile(ctx, job.RefFilename(), refbuf.Bytes()); err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "failed to put ref file",
				OriginError: err,
			}
		}
	}

	var jobbuf bytes.Buffer
	if err := utils.WriteJobTar(ctx, &jobbuf, contents, args); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "failed to write job tar",
			OriginError: err,
		}
	}
	if err := self.s3.PutFile(ctx, job.JobFilename(), jobbuf.Bytes()); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "failed to put job file",
			OriginError: err,
		}
	}

	urlTimeout := time.Duration(job.Timeout+job.GracePeriod+60) * time.Second
	job.RefFileURL, err = self.s3.GetFileURL(ctx, job.RefFilename(), urlTimeout)
	if err != nil {
		return types.FileNotExistError
	}
	job.JobFileURL, err = self.s3.GetFileURL(ctx, job.JobFilename(), urlTimeout)
	if err != nil {
		return types.FileNotExistError
	}

	return bunrouter.JSON(w, job)
}
