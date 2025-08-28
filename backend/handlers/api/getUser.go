package api

import (
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
)

// GetUser
//
//	@Param		csId	query		string	true	"User CsId to by queried"
//	@Success	200		{object}	map[string]string
//	@Failure	401		{object}	string	"not admin or not login"
//	@Router		/api/admin/user [get]
func GetUser(
	db interfaces.Database,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		ctx := req.Context()

		csId := req.URL.Query().Get("csId")

		user, err := db.GetUserResource(ctx, csId)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "query user resource failed",
				OriginError: err,
			}
		}

		return bunrouter.JSON(w, user)
	}
}
