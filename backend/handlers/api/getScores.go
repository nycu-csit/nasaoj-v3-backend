package api

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
)

// GetAdminScores
//
//	@Param		problemId	query		int								true	"Problem ID"
//	@Success	200			{object}	[]models.UserScoreWithStages	"Scores"
//	@Failure	400			{object}	string							"Invalid problem id"
//	@Router		/api/admin/scores [get]
func getAdminScoresPlaceholder() {}

// GetScores
//
//	@Param		problemId	query		int					true	"Problem ID"
//	@Success	200			{object}	[]models.UserScore	"Scores"
//	@Failure	400			{object}	string				"Invalid problem id"
//	@Router		/api/scores [get]
func GetScores(
	db interfaces.Database,
	isAdmin bool,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		ctx := req.Context()
		fmt.Fprintf(os.Stderr, "GET /api/scores\n")

		problemId, err := strconv.ParseInt(
			req.URL.Query().Get("problemId"), 10, 64)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     "invalid problem id",
				OriginError: err,
			}
		}

		if isAdmin {
			scores, err := db.GetScoresWithStages(ctx, problemId)
			if err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusInternalServerError,
					Message:     "score get failed",
					OriginError: err,
				}
			}

			w.WriteHeader(http.StatusOK)
			bunrouter.JSON(w, scores)
		} else {
			scores, err := db.GetScores(ctx, problemId)
			if err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusInternalServerError,
					Message:     "score get failed",
					OriginError: err,
				}
			}

			w.WriteHeader(http.StatusOK)
			bunrouter.JSON(w, scores)
		}
		return nil
	}
}
