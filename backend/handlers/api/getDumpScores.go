package api

import (
	"net/http"
	"strconv"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
)

// GetDumpScores
//
//	@Summary		Dump scores for a problem
//	@Description	Returns every user's score for a problem, honoring the deadline. Admin only.
//	@Param			problemId	path	int64	true	"problem id"
//	@Produce		json
//	@Success		200	{object}	[]models.UserScore	"scores"
//	@Router			/api/admin/dump-scores/{problemId} [get]
func (self *Handlers) GetDumpScores(
	w http.ResponseWriter, req bunrouter.Request,
) error {
	ctx := req.Context()

	problemId, err := strconv.ParseInt(
		req.Param("problemId"), 10, 64)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Message:     "failed to parse problem id",
			OriginError: err,
		}
	}

	scores, err := self.scores.GetScoresWithDeadline(ctx,
		middlewares.NamespaceIdFromContext(ctx), problemId)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "failed to get scores",
			OriginError: err,
		}
	}

	return bunrouter.JSON(w, scores)
}
