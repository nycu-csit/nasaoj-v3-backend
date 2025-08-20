package api

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type GetWorkerStatusesOutput struct {
	IP string `json:"ip"`

	models.WorkerStatus `json:",inline"`
}

// GetWorkerStatuses
//
//	@Produce	json
//	@Success	200	{object}	[]GetWorkerStatusesOutput
//	@Failure	401	{object}	string
//	@Router		/api/admin/worker-statuses [get]
func GetWorkerStatuses(
	workers interfaces.WorkerStatusStorage,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		ctx := req.Context()
		fmt.Fprintf(os.Stderr, "GET /api/admin/worker-statuses\n")

		res, err := workers.GetWorkers(ctx)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "worker status get failed",
				OriginError: err,
			}
		}

		ret := []GetWorkerStatusesOutput{}
		for ip, status := range res {
			if status.LastAccess < time.Now().Add(-time.Hour).Unix() {
				if err := workers.DeleteWorker(ctx, ip); err != nil {
					return middlewares.HTTPError{
						StatusCode:  http.StatusInternalServerError,
						Message:     "worker delete failed",
						OriginError: err,
					}
				}
				continue
			}
			ret = append(ret, GetWorkerStatusesOutput{
				IP:           ip,
				WorkerStatus: status,
			})
		}

		return bunrouter.JSON(w, ret)
	}
}
