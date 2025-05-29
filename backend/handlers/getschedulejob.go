package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

func GetScheduleJob(
	ctx context.Context,
	jobQue interfaces.JobQueue,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
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

		w.WriteHeader(http.StatusOK)
		return bunrouter.JSON(w, job)
	}
}
