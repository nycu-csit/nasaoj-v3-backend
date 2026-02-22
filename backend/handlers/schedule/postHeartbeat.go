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

	if err := self.workers.UpdateWorkerStatus(ctx,
		req.Header.Get("X-Worker-Name"),
		utils.GetRemoteAddr(req),
		heartbeat.Tags); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "worker status update failed",
			OriginError: err,
		}
	}

	return utils.Success(w)
}
