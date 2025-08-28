package api

import (
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

// DeleteUser
//
//	@Param		csId	query		string	true	"User CsId"
//	@Success	200		{object}	string
//	@Failure	400		{object}	string
//	@Failure	401		{object}	string
//	@Router		/api/admin/user [delete]
func DeleteUser(
	db interfaces.Database,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		ctx := req.Context()

		csId := req.URL.Query().Get("csId")
		if err := db.DeleteUser(ctx, csId); err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "user delete failed",
				OriginError: err,
			}
		}
		return utils.Success(w)
	}
}
