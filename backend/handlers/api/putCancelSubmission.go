package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

type PutCancelSubmissionInput struct {
	SubId int64 `json:"subId"`
}

// PutAdminCancelSubmission
//
//	@Param		request	body		PutCancelSubmissionInput	true	"query body"
//	@Success	200		{object}	string
//	@Failure	400		{object}	string
//	@Failure	401		{object}	string
//	@Router		/api/admin/submission/cancel [put]
func putAdminCancelSubmissionPlaceholder() {}

// PutCancelSubmission
//
//	@Param		request	body		PutCancelSubmissionInput	true	"query body"
//	@Success	200		{object}	string
//	@Failure	400		{object}	string
//	@Failure	401		{object}	string
//	@Failure	403		{object}	string
//	@Router		/api/submission/cancel [put]
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

		runningJobIds, err := self.submissions.CancelSubmission(ctx,
			input.SubId, ownerId)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusForbidden,
				Message:     "submission cancel failed",
				OriginError: err,
			}
		}

		// Add running jobs to cancelled set so workers can be notified
		if len(runningJobIds) > 0 {
			if err := self.workers.AddCancelledJobs(ctx, runningJobIds); err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusInternalServerError,
					Message:     "failed to add cancelled jobs",
					OriginError: err,
				}
			}
		}

		return utils.Success(w)
	}
}
