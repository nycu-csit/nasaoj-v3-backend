package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/uptrace/bun"
	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

// GetResults
//
//	@Param		problem	query		string				true	"Problem name"
//	@Success	200		{object}	models.Submission	"Last submission to the problem"
//	@Router		/api/results [get]
func GetResults(ctx context.Context, db *bun.DB) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		fmt.Fprintf(os.Stderr, "GET /api/results\n")

		problemName := req.URL.Query().Get("problem")

		resource, ok := req.Context().Value(
			middlewares.UserResourceType("")).(models.UserResource)
		if !ok {
			return fmt.Errorf("user resource missing\n")
		}

		var submission models.Submission
		err := db.NewSelect().Model(&submission).
			Where("csid = ?", resource.CsId).
			Where("problemname = ?", problemName).Limit(1).Scan(ctx)
		if err != nil {
			return fmt.Errorf("submission query failed, %s\n", err)
		}

		w.WriteHeader(http.StatusOK)
		bunrouter.JSON(w, submission)
		return nil
	}
}
