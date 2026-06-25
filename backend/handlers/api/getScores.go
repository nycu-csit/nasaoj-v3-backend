package api

import (
	"net/http"
	"strconv"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
)

// GetAdminScores
//
//	@Summary		Get scores (admin)
//	@Description	Returns all users' scores for a problem. Admin only.
//	@Param			problemId	query		int								true	"Problem ID"
//	@Success		200			{object}	[]models.UserScoreWithStages	"Scores"
//	@Failure		400			{object}	string							"Invalid problem id"
//	@Router			/api/admin/scores [get]
func getAdminScoresPlaceholder() {}

// GetScores
//
//	@Summary		Get scores
//	@Description	Returns the caller's score for a problem.
//	@Param			problemId	query		int					true	"Problem ID"
//	@Success		200			{object}	[]models.UserScore	"Scores"
//	@Failure		400			{object}	string				"Invalid problem id"
//	@Router			/api/scores [get]
func (self *Handlers) GetScores(
	isAdmin bool,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		ctx := req.Context()

		problemId, err := strconv.ParseInt(
			req.URL.Query().Get("problemId"), 10, 64)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     "invalid problem id",
				OriginError: err,
			}
		}

		namespaceId := middlewares.NamespaceIdFromContext(ctx)

		if isAdmin {
			scores, err := self.scores.GetScoresWithStages(ctx, namespaceId, problemId)
			if err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusInternalServerError,
					Message:     "score get failed",
					OriginError: err,
				}
			}

			return bunrouter.JSON(w, scores)
		} else {
			scores, err := self.scores.GetScores(ctx, namespaceId, problemId)
			if err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusInternalServerError,
					Message:     "score get failed",
					OriginError: err,
				}
			}

			return bunrouter.JSON(w, scores)
		}
	}
}
