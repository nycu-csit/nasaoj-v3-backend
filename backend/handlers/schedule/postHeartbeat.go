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
	Tags        []string `json:"tags"`
	Submissions []int64  `json:"submissions"`
}

type HeartbeatResponse struct {
	CancelSubmissions []int64 `json:"cancelSubmissions"`
}

// PostScheduleHeartbeat
//
//	@Summary		Worker heartbeat
//	@Description	Refreshes the worker's status and submission-queue lease, and
//	@Description	returns jobs / submissions the backend wants cancelled.
//	@Accept			json
//	@Produce		json
//	@Param			X-Worker-Name	header		string				true	"worker name"
//	@Param			request			body		HeartbeatRequest	true	"tags and active submissions"
//	@Success		200				{object}	HeartbeatResponse	"jobs/submissions to cancel"
//	@Failure		400				{object}	string				"bad request"
//	@Failure		500				{object}	string				"internal error"
//	@Router			/schedule/heartbeat [post]
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

	// Keep the submission-queue lease alive for the submissions this worker
	// still reports as running.
	if err := self.subQueue.RefreshLease(ctx,
		workerName, heartbeat.Submissions); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "failed to refresh submission lease",
			OriginError: err,
		}
	}

	// Determine which of the worker's active submissions have been cancelled.
	cancelSubs, err := self.workers.GetCancelledSubmissions(ctx, heartbeat.Submissions)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "failed to get cancelled submissions",
			OriginError: err,
		}
	}
	if len(cancelSubs) > 0 {
		if err := self.workers.RemoveCancelledSubmissions(ctx, cancelSubs); err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "failed to remove cancelled submissions",
				OriginError: err,
			}
		}
	}

	return bunrouter.JSON(w, HeartbeatResponse{
		CancelSubmissions: cancelSubs,
	})
}
