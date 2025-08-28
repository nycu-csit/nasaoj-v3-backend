package schedule

import (
	"net/http"
	"strings"
	"time"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func GetScheduleJob(
	db interfaces.Database,
	jobQue interfaces.JobQueue,
	workers interfaces.WorkerStatusStorage,
	s3 interfaces.ObjectStorage,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		ctx := req.Context()
		tags := req.URL.Query().Get("tags")

		if err := workers.UpdateWorkerStatus(ctx,
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
			job, err = jobQue.GetJob(ctx, tag)
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
		job.FileURL, err = s3.GetFileURL(ctx, job.Filename(), timeout)
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
		if err = db.UpdateJob(ctx, &job); err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusServiceUnavailable,
				Message:     "job update failed",
				OriginError: err,
			}
		}

		return bunrouter.JSON(w, job)
	}
}
