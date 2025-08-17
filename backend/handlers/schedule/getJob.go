package schedule

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

func GetScheduleJob(
	db interfaces.Database,
	jobQue interfaces.JobQueue,
	s3 interfaces.ObjectStorage,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		ctx := req.Context()
		tags := req.URL.Query().Get("tags")
		fmt.Fprintf(os.Stderr, "GET /schedule/job?tags=%s\n", tags)

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
				StatusCode:  http.StatusServiceUnavailable,
				Message:     "no job left in queue",
				OriginError: err,
			}
		}

		timeout := time.Duration(job.HardTL) * time.Second
		job.FileURL, err = s3.GetFileURL(ctx, job.Filename(), timeout)
		if err != nil {
			return fmt.Errorf("get file url for '%s' failed, %w", job.Filename(), err)
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

		w.WriteHeader(http.StatusOK)
		return bunrouter.JSON(w, job)
	}
}
