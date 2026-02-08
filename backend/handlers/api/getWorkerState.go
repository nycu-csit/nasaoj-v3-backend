package api

import (
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type GetWorkerStatesOutput struct {
	Name string   `json:"name"`
	Jobs []string `json:"jobs"`

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
			Message:     "failed to get worker status",
			OriginError: err,
		}
	}

	ret := []GetWorkerStatesOutput{}
	for name, status := range res {
		jobIds, err := self.workers.GetWorkerJobs(ctx, name)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "failed to get worker jobs",
				OriginError: err,
			}
		}
		ret = append(ret, GetWorkerStatesOutput{
			Name:         name,
			Jobs:         jobIds,
			WorkerStatus: status,
		})
	}

	return bunrouter.JSON(w, ret)
}
