package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/workflows"
)

func PutScheduleResult(
	ctx context.Context,
	db interfaces.Database,
	jobQue interfaces.JobQueue,
	s3 interfaces.ObjectStorage,
	problems interfaces.ProblemStorage,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		fmt.Fprintf(os.Stderr, "PUT /schedule/result\n")

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

		// Handle hard timeout if timeout isn't put on worker
		if time.Now().Compare(time.Unix(job.Deadline, 0)) > 0 {
			job.Status = models.StatusTimeout
		}

		if job.Status != models.StatusSuccess {
			if err = db.SkipJobs(ctx, job.SubId); err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusInternalServerError,
					Message:     "job skip status update failed",
					OriginError: err,
				}
			}
		}
		if err = db.UpdateJob(ctx, &job); err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "job update failed",
				OriginError: err,
			}
		}

		err = s3.PutFile(ctx, job.StdoutKey(), []byte(job.Stdout))
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "stdout file put failed",
				OriginError: err,
			}
		}

		err = s3.PutFile(ctx, job.StderrKey(), []byte(job.Stderr))
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "stderr file put failed",
				OriginError: err,
			}
		}

		if job.Status == models.StatusSuccess {
			err = workflows.HandleStageDone(ctx, db, jobQue, s3,
				problems, job)
			if err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusInternalServerError,
					Message:     "handle job done failed",
					OriginError: err,
				}
			}
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "success")
		return nil
	}
}
