package schedule

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/workflows"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func (self *Handlers) PutScheduleResult(
	w http.ResponseWriter, req bunrouter.Request,
) error {
	ctx := req.Context()

	if err := self.workers.UpdateLastAccess(ctx,
		req.Header.Get("X-Worker-Name")); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "failed to update worker last access",
			OriginError: err,
		}
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "data retrieve failed",
			OriginError: err,
		}
	}

	var job models.Job
	err = json.Unmarshal(body, &job)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Message:     "json unmarshal failed",
			OriginError: err,
		}
	}

	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.Int64("submission.id", job.SubId),
		attribute.String("stage.name", job.StageName),
		attribute.String("testcase.name", job.TestName))

	// Handle hard timeout if timeout isn't put on worker
	if time.Now().Compare(time.Unix(job.Deadline, 0)) > 0 {
		tracing.Logger.Ctx(ctx).Warn("job exceeded hard deadline",
			zap.String("deadline", time.Unix(job.Deadline, 0).Format(time.RFC3339)),
			zap.String("now", time.Now().Format(time.RFC3339)))
		job.Status = models.StatusTimeout
	}

	if job.Status != models.StatusSuccess {
		if err = self.jobs.SkipJobs(ctx, job.SubId); err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "failed to update job skip status",
				OriginError: err,
			}
		}
	}
	if err = self.jobs.UpdateJob(ctx, &job); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "failed to update job",
			OriginError: err,
		}
	}

	err = self.s3.PutFile(ctx, job.StdoutKey(), []byte(job.Stdout))
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "stdout file put failed",
			OriginError: err,
		}
	}

	err = self.s3.PutFile(ctx, job.StderrKey(), []byte(job.Stderr))
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "stderr file put failed",
			OriginError: err,
		}
	}

	submission, err := self.submissions.GetSubmission(ctx, job.SubId)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "failed to get submission",
			OriginError: err,
		}
	}

	if job.Status == models.StatusSuccess &&
		submission.Type == models.RefTypeProblem {
		err = workflows.HandleJobDone(ctx, self.db, self.jobQue, self.s3,
			self.problems, self.contents, self.jobs, self.tokens,
			submission, job)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "handle job done failed",
				OriginError: err,
			}
		}
	}

	return utils.Success(w)
}
