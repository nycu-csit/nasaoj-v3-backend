package schedule

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/workflows"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// GetSubmissionOutput is returned to a worker that claims a submission. It
// carries the submission metadata and the (dehydrated) problem/tool definition
// the worker needs to resolve stage dependencies. The per-job scripts and tar
// files are fetched lazily through POST /schedule/job/marshal.
type GetSubmissionOutput struct {
	Submission models.Submission `json:"submission"`
	Problem    models.Problem    `json:"problem"`
	Tool       models.Tool       `json:"tool"`
}

// GetScheduleSubmission
//
//	@Summary		Claim a queued submission
//	@Description	Claims one queued submission for the requesting worker,
//	@Description	marks its jobs Blocked, and returns the submission with its
//	@Description	(dehydrated) problem/tool definition.
//	@Produce		json
//	@Param			tags			query		string				false	"comma-separated worker tags"
//	@Param			X-Worker-Name	header		string				true	"worker name"
//	@Success		200				{object}	GetSubmissionOutput	"claimed submission"
//	@Success		204				{object}	string				"no submission in queue"
//	@Failure		503				{object}	string				"claim failed"
//	@Router			/schedule/submission [get]
func (self *Handlers) GetScheduleSubmission(
	w http.ResponseWriter, req bunrouter.Request,
) error {
	ctx := req.Context()
	tagList := strings.Split(req.URL.Query().Get("tags"), ",")
	workerName := req.Header.Get("X-Worker-Name")

	item, err := self.subQueue.ClaimNext(ctx, workerName, tagList)
	if errors.Is(err, sql.ErrNoRows) {
		return middlewares.HTTPError{
			StatusCode: http.StatusNoContent,
			Message:    "no submission left in queue",
		}
	}
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusServiceUnavailable,
			Message:     "failed to claim submission",
			OriginError: err,
		}
	}

	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.Int64("submission.id", item.SubmissionId),
		attribute.String("worker.name", workerName))

	// The submission is now assigned to this worker: move its queued jobs to
	// Blocked (they become Running individually as the worker marshals them).
	if err := self.jobs.BlockSubmissionJobs(ctx, item.SubmissionId); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "failed to block submission jobs",
			OriginError: err,
		}
	}

	// A submission with no jobs (or one whose jobs are all already terminal,
	// e.g. cancelled before being claimed) is done immediately.
	if err := workflows.CompleteIfDone(ctx,
		self.submissions, self.subQueue, item.SubmissionId); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "failed to complete submission",
			OriginError: err,
		}
	}

	submission, err := self.submissions.GetSubmission(ctx, item.SubmissionId)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "failed to get submission",
			OriginError: err,
		}
	}

	resp := GetSubmissionOutput{Submission: submission}
	switch submission.Type {
	case models.RefTypeProblem:
		problem, err := self.problems.GetProblem(ctx, submission.ProblemSHA1)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to get problem for claimed submission",
					zap.String("problem.sha1", submission.ProblemSHA1),
					zap.Error(err))
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "failed to get problem",
				OriginError: err,
			}
		}
		resp.Problem = problem
	case models.RefTypeTool:
		tool, err := self.tools.GetTool(ctx, submission.ToolSHA1)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to get tool for claimed submission",
					zap.String("tool.sha1", submission.ToolSHA1),
					zap.Error(err))
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "failed to get tool",
				OriginError: err,
			}
		}
		resp.Tool = tool
	}

	return bunrouter.JSON(w, resp)
}
