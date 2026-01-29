package api

import (
	"net/http"
	"time"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type GetWorkerStatesOutput struct {
	IP string `json:"ip"`

	models.WorkerStatus `json:",inline"`
}

// GetWorkerState
//
//	@Produce	json
//	@Success	200	{object}	[]GetWorkerStatesOutput
//	@Failure	401	{object}	string
//	@Router		/api/admin/worker/state [get]
func (self *Handlers) GetWorkerStates(
	w http.ResponseWriter, req bunrouter.Request,
) error {
	ctx := req.Context()

	res, err := self.workers.GetWorkers(ctx)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "worker status get failed",
			OriginError: err,
		}
	}

	ret := []GetWorkerStatesOutput{}
	for ip, status := range res {
		if status.LastAccess < time.Now().Add(-time.Hour).Unix() {
			if err := self.workers.DeleteWorker(ctx, ip); err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusInternalServerError,
					Message:     "worker delete failed",
					OriginError: err,
				}
			}
			continue
		}
		ret = append(ret, GetWorkerStatesOutput{
			IP:           ip,
			WorkerStatus: status,
		})
	}

	return bunrouter.JSON(w, ret)
}
