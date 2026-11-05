package api

import (
	"net/http"
	"slices"
	"strconv"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

// DeleteSubmission
//
//	@Summary		Delete a submission
//	@Description	Deletes a single submission and its jobs by id. Admin only.
//	@Param			subId	query		int64	true	"Submission Id"
//	@Success		200		{object}	string
//	@Failure		400		{object}	string
//	@Failure		401		{object}	string
//	@Router			/api/admin/submission [delete]
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

	submission, err := self.submissions.GetSubmission(ctx, subId)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusNotFound,
			Message:     "submission not found",
			OriginError: err,
		}
	}
	if !slices.Contains(
		middlewares.NamespaceIdsFromContext(ctx), submission.NamespaceId) {
		return middlewares.HTTPError{
			StatusCode: http.StatusNotFound,
			Message:    "submission not in namespace",
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
