package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

type ScoreRequest struct {
	SubId int64 `json:"subId"`
	Score int64 `json:"score"`
}

// PutScore
//
//	@Summary		Update submission score for self-managed scoring
//	@Description	Updates the score for a submission that uses self-managed scoring type
//	@Accept			json
//	@Produce		json
//	@Param			body	body		ScoreRequest	true	"Score update request"
//	@Success		200		{object}	string			"Success"
//	@Failure		400		{object}	string			"Invalid request or problem does not use self-managed scoring"
//	@Failure		404		{object}	string			"Submission not found"
//	@Failure		500		{object}	string			"Internal server error"
//	@Router			/api/score [put]
func (self *Handlers) PutScore(
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

	var scoreReq ScoreRequest
	err = json.Unmarshal(body, &scoreReq)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Message:     "json unmarshal failed",
			OriginError: err,
		}
	}

	// Update submission score
	err = self.submissions.UpdateSubmissionScore(ctx,
		scoreReq.SubId, scoreReq.Score)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "failed to update score",
			OriginError: err,
		}
	}

	return utils.Success(w)
}
