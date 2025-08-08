package api

import (
	"fmt"
	"net/http"
	"os"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
)

// GetUsers
//
//	@Success	200	{object}	[]models.UserResource
//	@Failure	401	{object}	string	"not admin or not login"
//	@Router		/api/admin/users [get]
func GetUsers(
	db interfaces.Database,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		ctx := req.Context()
		fmt.Fprintf(os.Stderr, "GET /api/admin/users\n")

		users, err := db.GetUsers(ctx)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "query user resource failed",
				OriginError: err,
			}
		}

		bunrouter.JSON(w, users)
		return nil
	}
}
