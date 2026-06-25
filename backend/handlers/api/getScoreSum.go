package api

import (
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
)

// GetScoreSum
//
//	@Summary		Sum of scores in a namespace
//	@Description	Per-user total score across all enabled problems in the
//	@Description	namespace (from the X-Namespace header), sorted descending.
//	@Produce		json
//	@Param			X-Namespace	header		string						false	"namespace name or id"
//	@Success		200			{object}	[]models.ScoreboardEntry	"score sums"
//	@Failure		401			{object}	string
//	@Failure		403			{object}	string
//	@Router			/api/scores/sum [get]
func (self *Handlers) GetScoreSum(
	w http.ResponseWriter, req bunrouter.Request,
) error {
	ctx := req.Context()

	board, err := self.scores.GetNamespaceScoreboard(ctx,
		middlewares.NamespaceIdFromContext(ctx))
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "failed to get score sum",
			OriginError: err,
		}
	}

	return bunrouter.JSON(w, board)
}
