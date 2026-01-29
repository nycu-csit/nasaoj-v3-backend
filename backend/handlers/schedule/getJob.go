package schedule

import (
	"net/http"
	"strings"
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

func (self *Handlers) GetScheduleJob(
	w http.ResponseWriter, req bunrouter.Request,
) error {
	ctx := req.Context()
	tags := req.URL.Query().Get("tags")

	if err := self.workers.UpdateWorkerStatus(ctx,
		utils.GetRemoteAddr(req),
		strings.Split(tags, ",")); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "worker status update failed",
			OriginError: err,
		}
	}

	// Get Job from queue
	var err error
	var job models.Job
	for _, tag := range strings.Split(tags, ",") {
		job, err = self.jobQue.GetJob(ctx, tag)
		if err == nil {
			break
		}
	}
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusNoContent,
			Message:     "no job left in queue",
			OriginError: err,
		}
	}

	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.Int64("submission.id", job.SubId),
		attribute.String("job.stage", job.StageName),
		attribute.String("job.test", job.TestName))

	timeout := time.Duration(job.HardTL) * time.Second
	job.FileURL, err = self.s3.GetFileURL(ctx, job.Filename(), timeout)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to get file url",
				zap.String("file.name", job.Filename()),
				zap.Error(err))
		return types.FileNotExistError
	}

	job.Deadline = time.Now().
		Add(time.Duration(job.HardTL) * time.Second).Unix()
	job.Status = models.StatusRunning
	if err = self.jobs.UpdateJob(ctx, &job); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusServiceUnavailable,
			Message:     "job update failed",
			OriginError: err,
		}
	}

	return bunrouter.JSON(w, job)
}
