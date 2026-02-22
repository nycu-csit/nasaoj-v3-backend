package api

import (
	"net/http"
	"strconv"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

// PutAdminCancelSubmission
//
//	@Param		subId	query		int64	true	"Submission Id"
//	@Success	200		{object}	string
//	@Failure	400		{object}	string
//	@Failure	401		{object}	string
//	@Router		/api/admin/submission/cancel [put]
func putAdminCancelSubmissionPlaceholder() {}

// PutCancelSubmission
//
//	@Param		subId	query		int64	true	"Submission Id"
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

		// Get problem tag for removing jobs from queue
		var tag string
		switch submission.Type {
		case models.RefTypeProblem:
			problem, err := self.problems.GetProblem(ctx, submission.ProblemSHA1)
			if err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusInternalServerError,
					Message:     "problem not found",
					OriginError: err,
				}
			}
			tag = problem.Tag
		case models.RefTypeTool:
			tool, err := self.tools.GetTool(ctx, submission.ToolSHA1)
			if err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusInternalServerError,
					Message:     "tool not found",
					OriginError: err,
				}
			}
			tag = tool.Tag
		}

		// Remove pending jobs from the queue
		for _, job := range submission.Jobs {
			if job.Status == models.StatusPending {
				_ = self.jobQue.RemoveJob(ctx, job.Id(), tag)
			}
		}

		if err := self.submissions.CancelSubmission(ctx, subId, ownerId); err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusForbidden,
				Message:     "submission cancel failed",
				OriginError: err,
			}
		}

		return utils.Success(w)
	}
}
