package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

// PostReloadScores
//
//	@Success	200	{string} string "Success"
//	@Failure	401
//	@Router		/api/admin/reloadscores [post]
func PostReloadScores(
	ctx context.Context,
	db interfaces.Database,
	scoreCache interfaces.ScoreCache,
	problems utils.ProblemRepository,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		fmt.Fprintf(os.Stderr, "POST /api/admin/reloadscores\n")

		err := scoreCache.Reload(ctx, db, problems)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "reload scores failed",
				OriginError: err,
			}
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Success")
		return nil
	}
}
