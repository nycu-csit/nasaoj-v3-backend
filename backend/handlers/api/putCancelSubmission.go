package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/workflows"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

type PutCancelSubmissionInput struct {
	SubId int64 `json:"subId"`
}

// PutAdminCancelSubmission
//
//	@Summary		Cancel a submission (admin)
//	@Description	Cancels any in-flight submission and signals workers to stop it. Admin only.
//	@Param			request	body		PutCancelSubmissionInput	true	"query body"
//	@Success		200		{object}	string
//	@Failure		400		{object}	string
//	@Failure		401		{object}	string
//	@Router			/api/admin/submission/cancel [put]
func putAdminCancelSubmissionPlaceholder() {}

// PutCancelSubmission
//
//	@Summary		Cancel a submission
//	@Description	Cancels one of the caller's in-flight submissions.
//	@Param			request	body		PutCancelSubmissionInput	true	"query body"
//	@Success		200		{object}	string
//	@Failure		400		{object}	string
//	@Failure		401		{object}	string
//	@Failure		403		{object}	string
//	@Router			/api/submission/cancel [put]
func (self *Handlers) PutCancelSubmission(
	isAdmin bool,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		ctx := req.Context()

		b, err := io.ReadAll(req.Body)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     "failed to read payload",
				OriginError: err,
			}
		}

		var input PutCancelSubmissionInput
		if err := json.Unmarshal(b, &input); err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     "failed to unmarshal json",
				OriginError: err,
			}
		}

		submission, err := self.submissions.GetSubmission(ctx, input.SubId)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     "submission not found",
				OriginError: err,
			}
		}

		var ownerId int64
		if isAdmin {
			ownerId = submission.OwnerId
		} else {
			privs, ok := ctx.Value(types.Privileges("")).(models.Privileges)
			if !ok {
				return middlewares.HTTPError{
					StatusCode: http.StatusUnauthorized,
					Message:    "privs not exist",
				}
			}
			ownerId = privs.OwnerId
		}

		if err := self.submissions.CancelSubmission(ctx,
			input.SubId, ownerId); err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusForbidden,
				Message:     "submission cancel failed",
				OriginError: err,
			}
		}

		// Mark the whole submission cancelled. The owning worker aborts it on
		// the next heartbeat, which cancels its in-flight jobs too.
		if err := self.workers.AddCancelledSubmissions(ctx,
			[]int64{input.SubId}); err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "failed to add cancelled submission",
				OriginError: err,
			}
		}

		// If no jobs remain running (e.g. the submission was still queued),
		// it is already done and can be removed from the queue now. Otherwise
		// the running jobs' results complete it.
		if err := workflows.CompleteIfDone(ctx,
			self.submissions, self.subQueue, input.SubId); err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "failed to complete submission",
				OriginError: err,
			}
		}

		return utils.Success(w)
	}
}
