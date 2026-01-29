package api

import (
	"net/http"
	"strconv"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

// DeleteUser
//
//	@Param		csId	query		string	true	"User CsId"
//	@Param		prune	query		bool	false	"If delete submissions"	attribute(optional)
//	@Success	200		{object}	string
//	@Failure	400		{object}	string
//	@Failure	401		{object}	string
//	@Router		/api/admin/user [delete]
func (self *Handlers) DeleteUser(
	w http.ResponseWriter, req bunrouter.Request,
) error {
	ctx := req.Context()

	csId := req.URL.Query().Get("csId")
	prune, err := strconv.ParseBool(req.URL.Query().Get("prune"))
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Message:     "failed to parse param 'prune'",
			OriginError: err,
		}
	}

	if prune {
		err := self.submissions.DeleteUserAndSubmissions(ctx, csId)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "failed to delete user and submissions",
				OriginError: err,
			}
		}
	} else {
		if err := self.db.DeleteUser(ctx, csId); err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "failed to delete user",
				OriginError: err,
			}
		}
	}
	return utils.Success(w)
}
