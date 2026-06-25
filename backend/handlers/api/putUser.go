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

// PutUserInput is sent by a worker forwarding a job's put-args. `SubId`
// identifies the submission (for the ownership check and to resolve the target
// user); `Args` are the user args to merge.
type PutUserInput struct {
	SubId int64             `json:"subId"`
	Args  map[string]string `json:"args"`
}

// PutUser
//
//	@Summary		Put user args
//	@Description	Merges args into a user's resource; used by jobs via the worker to persist outputs.
//	@Param			request	body	PutUserInput	true	"User resource"
//	@Accept			json
//	@Success		200	{object}	string	"success"
//	@Failure		401	{object}	string
//	@Router			/api/user [put]
func (self *Handlers) PutUser(
	w http.ResponseWriter, req bunrouter.Request,
) error {
	ctx := req.Context()

	privs, ok := ctx.Value(types.Privileges("")).(models.Privileges)
	if !ok {
		return middlewares.HTTPError{
			StatusCode: http.StatusUnauthorized,
			Message:    "privs not exist",
		}
	}

	b, err := io.ReadAll(req.Body)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Message:     "read payload failed",
			OriginError: err,
		}
	}

	var input PutUserInput
	if err := json.Unmarshal(b, &input); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Message:     "json unmarshal failed",
			OriginError: err,
		}
	}

	// Resolve the target user. A worker forwards on behalf of a job: verify the
	// submission is owned by that worker, then derive the owner from it. Direct
	// callers (legacy) update themselves.
	ownerId := privs.OwnerId
	if workerName := req.Header.Get("X-Worker-Name"); workerName != "" {
		owner, err := self.subQueue.SubmissionWorker(ctx, input.SubId)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "failed to check submission owner",
				OriginError: err,
			}
		}
		if owner != workerName {
			return middlewares.HTTPError{
				StatusCode: http.StatusForbidden,
				Message:    "submission not owned by this worker",
			}
		}
		submission, err := self.submissions.GetSubmission(ctx, input.SubId)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "failed to get submission",
				OriginError: err,
			}
		}
		ownerId = submission.OwnerId
	}

	if err := self.db.UpdateUserArgsById(ctx,
		models.UserResource{
			Id:   ownerId,
			Args: input.Args,
		}); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "insert user resources failed",
			OriginError: err,
		}
	}

	return utils.Success(w)
}
