package api

import (
	"net/http"
	"strconv"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

// DeleteSubmission
//
//	@Param		subId	query		int64	true	"Submission Id"
//	@Success	200		{object}	string
//	@Failure	400		{object}	string
//	@Failure	401		{object}	string
//	@Router		/api/admin/submission [delete]
func (self *Handlers) DeleteSubmission(
	w http.ResponseWriter, req bunrouter.Request,
) error {
	ctx := req.Context()

	subId, err := strconv.ParseInt(req.URL.Query().Get("subId"), 10, 64)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Message:     "submission id parse failed",
			OriginError: err,
		}
	}

	if err := self.submissions.DeleteSubmission(ctx, subId); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "submission delete failed",
			OriginError: err,
		}
	}
	return utils.Success(w)
}
