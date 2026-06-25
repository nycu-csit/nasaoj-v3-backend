package api

import (
	"net/http"
	"strconv"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
)

// GetUser
//
//	@Summary		Get a user (admin)
//	@Description	Returns a single user's profile by id. Admin only.
//	@Param			id	query		int64	true	"User Id to be queried"
//	@Success		200	{object}	map[string]string
//	@Failure		401	{object}	string	"not admin or not login"
//	@Router			/api/admin/user [get]
func (self *Handlers) GetUser(
	w http.ResponseWriter, req bunrouter.Request,
) error {
	ctx := req.Context()

	id, err := strconv.ParseInt(req.URL.Query().Get("id"), 10, 64)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Message:     "failed to parse param 'id'",
			OriginError: err,
		}
	}

	user, err := self.db.GetUserResource(ctx, id)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "query user resource failed",
			OriginError: err,
		}
	}

	return bunrouter.JSON(w, user)
}
