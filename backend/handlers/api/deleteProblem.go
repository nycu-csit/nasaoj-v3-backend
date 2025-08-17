package api

import (
	"fmt"
	"net/http"
	"os"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
)

// DeleteProblem
//
//	@Param		problemSHA1	query		string	true	"Problem SHA1"
//	@Success	200			{object}	string
//	@Failure	400			{object}	string
//	@Failure	401			{object}	string
//	@Router		/api/admin/problem [delete]
func DeleteProblem(
	db interfaces.Database,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		ctx := req.Context()
		fmt.Fprintf(os.Stderr, "DELETE /api/admin/problem\n")

		problemSHA1 := req.URL.Query().Get("problemSHA1")
		if err := db.DeleteProblemAndSubmissions(ctx, problemSHA1); err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "problem delete failed",
				OriginError: err,
			}
		}
		fmt.Fprintf(w, "success")
		return nil
	}
}
