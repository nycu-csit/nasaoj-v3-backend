package api

import (
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
)

// GetUsers
//
//	@Summary		List users (admin)
//	@Description	Returns all users. Admin only.
//	@Success		200	{object}	[]models.UserResource
//	@Failure		401	{object}	string	"not admin or not login"
//	@Router			/api/admin/users [get]
func (self *Handlers) GetUsers(
	w http.ResponseWriter, req bunrouter.Request,
) error {
	ctx := req.Context()

	users, err := self.db.GetUsers(ctx)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "query user resource failed",
			OriginError: err,
		}
	}

	return bunrouter.JSON(w, users)
}
