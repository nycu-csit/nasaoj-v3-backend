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
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func (self *Handlers) GetScheduleJob(
	w http.ResponseWriter, req bunrouter.Request,
) error {
	ctx := req.Context()
	tags := req.URL.Query().Get("tags")
	tagList := strings.Split(tags, ",")

	var err error
	var job models.Job
	var jobFound bool

	for i := 0; !jobFound; i++ {
		var foundInThisIteration bool
		var currentTag string

		for _, tag := range tagList {
			if i%2 == 0 {
				job, err = self.jobQue.GetRetryJob(ctx, tag)
				if err == nil {
					foundInThisIteration = true
					currentTag = tag
					break
				}
			} else {
				job, err = self.jobQue.GetJob(ctx, tag)
				if err == nil {
					foundInThisIteration = true
					currentTag = tag
					break
				}
			}
		}

		if !foundInThisIteration {
			if i < 2 {
				continue
			}
			return middlewares.HTTPError{
				StatusCode:  http.StatusNoContent,
				Message:     "no job left in queue",
				OriginError: err,
			}
		}

		if err = self.jobQue.Lock(ctx, &job); err != nil {
			// Lock acquisition failed - put job in retry queue
			tracing.Logger.Ctx(ctx).
				Debug("failed to acquire resource group lock, putting to retry queue",
					zap.String("job.id", job.Id()),
					zap.String("resourceGroup", job.ResourceGroup),
					zap.Error(err))

			if putErr := self.jobQue.PutRetryJob(ctx, job, currentTag); putErr != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to put job to retry queue",
						zap.String("job.id", job.Id()),
						zap.Error(putErr))
			}
			continue
		}

		// Lock acquired (or no lock needed)
		jobFound = true
	}

	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.Int64("submission.id", job.SubId),
		attribute.String("job.stage", job.StageName),
		attribute.String("job.test", job.TestName))

	timeout := time.Duration(job.Timeout+1) * time.Second
	job.RefFileURL, err = self.s3.GetFileURL(ctx, job.RefFilename(), timeout)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to get ref file url",
				zap.String("refFile.name", job.RefFilename()),
				zap.Error(err))
		return types.FileNotExistError
	}
	job.JobFileURL, err = self.s3.GetFileURL(ctx, job.JobFilename(), timeout)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to get job file url",
				zap.String("jobFile.name", job.JobFilename()),
				zap.Error(err))
		return types.FileNotExistError
	}

	// Add worker job tracking BEFORE setting status to Running
	// This prevents a race condition with CleanOrphanedJobs cron
	workerName := req.Header.Get("X-Worker-Name")
	if err = self.workers.AddWorkerJob(ctx, workerName, job.Id()); err != nil {
		tracing.Logger.Ctx(ctx).
			Warn("failed to add worker job",
				zap.String("worker", workerName),
				zap.String("job.id", job.Id()),
				zap.Error(err))
	}

	job.Status = models.StatusRunning
	if err = self.jobs.UpdateJob(ctx, &job, models.StatusPending); err != nil {
		// Rollback worker job tracking on failure
		_ = self.workers.RemoveWorkerJob(ctx, workerName, job.Id())
		return middlewares.HTTPError{
			StatusCode:  http.StatusServiceUnavailable,
			Message:     "job update failed",
			OriginError: err,
		}
	}

	return bunrouter.JSON(w, job)
}
