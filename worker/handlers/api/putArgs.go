package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/worker/middlewares"
)

type putUserForward struct {
	SubId int64             `json:"subId"`
	Args  map[string]string `json:"args"`
}

// PutArgs
//
//	@Summary		Put user args on behalf of a job
//	@Description	Forwards a job's put-args to the backend, tagging it with the
//	@Description	subId from the verified jobToken.
//	@Accept			json
//	@Produce		json
//	@Param			request	body		map[string]string	true	"args to merge"
//	@Success		200		{object}	string				"success"
//	@Failure		401		{object}	string				"invalid job token"
//	@Failure		403		{object}	string				"submission not owned by this worker"
//	@Failure		502		{object}	string				"backend request failed"
//	@Router			/args [put]
func (self *Handlers) PutArgs(
	w http.ResponseWriter, req bunrouter.Request,
) error {
	claims, ok := jobClaims(req.Request)
	if !ok {
		return middlewares.HTTPError{
			StatusCode: http.StatusUnauthorized,
			Message:    "job claims not exist",
		}
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Message:     "read payload failed",
			OriginError: err,
		}
	}

	var args map[string]string
	if err := json.Unmarshal(body, &args); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Message:     "json unmarshal failed",
			OriginError: err,
		}
	}

	resp, err := self.client.R().
		SetContext(req.Context()).
		SetBody(putUserForward{SubId: claims.SubId, Args: args}).
		Put("/api/user")
	return self.forward(w, resp, err)
}
