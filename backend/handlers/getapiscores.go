package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

// GetAdminScores
//
//	@Param		problemId	query		int					true	"Problem ID"
//	@Success	200			{object}	[]models.UserScore	"Scores"
//	@Failure	400			{object}	string				"Invalid problem id"
//	@Router		/api/admin/scores [get]
func getAdminScoresPlaceholder() {}

// GetScores
//
//	@Param		problemId	query		int					true	"Problem ID"
//	@Success	200			{object}	[]models.UserScore	"Scores"
//	@Failure	400			{object}	string				"Invalid problem id"
//	@Router		/api/scores [get]
func GetScores(
	ctx context.Context,
	scoreCache interfaces.ScoreCache,
	problems utils.ProblemRepository,
	isAdmin bool,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
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

		problem, ok := problems[problemId]
		if !ok {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     "invalid problem id",
				OriginError: err,
			}
		}

		scores, err := scoreCache.Get(ctx, problem, isAdmin)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "score cache get failed",
				OriginError: err,
			}
		}

		w.WriteHeader(http.StatusOK)
		bunrouter.JSON(w, scores)
		return nil
	}
}
