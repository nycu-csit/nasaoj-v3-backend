package schedule

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

type HeartbeatRequest struct {
	Tags []string `json:"tags"`
}

type HeartbeatResponse struct {
	CancelJobs []string `json:"cancelJobs"`
}

func (self *Handlers) PostScheduleHeartbeat(
	w http.ResponseWriter, req bunrouter.Request,
) error {
	ctx := req.Context()

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "data retrieve failed",
			OriginError: err,
		}
	}

	var heartbeat HeartbeatRequest
	if err := json.Unmarshal(body, &heartbeat); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Message:     "json unmarshal failed",
			OriginError: err,
		}
	}

	workerName := req.Header.Get("X-Worker-Name")

	if err := self.workers.UpdateWorkerStatus(ctx,
		workerName,
		utils.GetRemoteAddr(req),
		heartbeat.Tags); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "worker status update failed",
			OriginError: err,
		}
	}

	// Get jobs running on this worker
	workerJobs, err := self.workers.GetWorkerJobs(ctx, workerName)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "failed to get worker jobs",
			OriginError: err,
		}
	}

	// Check which jobs should be cancelled
	cancelJobs, err := self.workers.GetCancelledJobs(ctx, workerJobs)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "failed to get cancelled jobs",
			OriginError: err,
		}
	}

	// Remove cancelled jobs from the set (they will be handled by the worker)
	if len(cancelJobs) > 0 {
		if err := self.workers.RemoveCancelledJobs(ctx, cancelJobs); err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "failed to remove cancelled jobs",
				OriginError: err,
			}
		}
	}

	return bunrouter.JSON(w, HeartbeatResponse{CancelJobs: cancelJobs})
}
