package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/worker/middlewares"
)

// ScoreInput is the job-facing body: only the score. The submission is
// identified by the jobToken, not the request.
type ScoreInput struct {
	Score int64 `json:"score"`
}

// scoreForward is the backend-facing body, tagged with the subId from the
// verified jobToken.
type scoreForward struct {
	SubId int64 `json:"subId"`
	Score int64 `json:"score"`
}

// PutScore
//
//	@Summary		Put a submission score on behalf of a job
//	@Description	Forwards a job's put-score to the backend. The submission is
//	@Description	taken from the verified jobToken, not the request body.
//	@Accept			json
//	@Produce		json
//	@Param			request	body		ScoreInput	true	"score"
//	@Success		200		{object}	string		"success"
//	@Failure		401		{object}	string		"invalid job token"
//	@Failure		403		{object}	string		"submission not owned by this worker"
//	@Failure		502		{object}	string		"backend request failed"
//	@Router			/score [put]
func (self *Handlers) PutScore(
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

	var in ScoreInput
	if err := json.Unmarshal(body, &in); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Message:     "json unmarshal failed",
			OriginError: err,
		}
	}

	resp, err := self.client.R().
		SetContext(req.Context()).
		SetBody(scoreForward{SubId: claims.SubId, Score: in.Score}).
		Put("/api/score")
	return self.forward(w, resp, err)
}
