package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

type DeleteNamespaceUsersInput struct {
	UserIds []int64 `json:"userIds"`
}

// DeleteNamespaceUsers
//
//	@Summary		Remove namespace members
//	@Description	Removes users from the namespace in the X-Namespace header.
//	@Description	Global admin or admin of that namespace.
//	@Accept			json
//	@Produce		json
//	@Param			X-Namespace	header		string						true	"namespace name or id"
//	@Param			request		body		DeleteNamespaceUsersInput	true	"members"
//	@Success		200			{object}	string
//	@Failure		400			{object}	string
//	@Failure		403			{object}	string
//	@Router			/api/admin/namespace/users [delete]
func (self *Handlers) DeleteNamespaceUsers(
	w http.ResponseWriter, req bunrouter.Request,
) error {
	ctx := req.Context()
	namespaceId := middlewares.NamespaceIdFromContext(ctx)

	b, err := io.ReadAll(req.Body)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Message:     "read payload failed",
			OriginError: err,
		}
	}

	var input DeleteNamespaceUsersInput
	if err := json.Unmarshal(b, &input); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Message:     "json unmarshal failed",
			OriginError: err,
		}
	}

	if err := self.namespaces.RemoveNamespaceMembers(
		ctx, namespaceId, input.UserIds); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "failed to remove namespace members",
			OriginError: err,
		}
	}

	return utils.Success(w)
}
