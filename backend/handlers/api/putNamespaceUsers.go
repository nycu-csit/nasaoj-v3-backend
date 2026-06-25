package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

type PutNamespaceUsersInput struct {
	UserIds []int64 `json:"userIds"`
	IsAdmin bool    `json:"isAdmin"`
}

// PutNamespaceUsers
//
//	@Summary		Add namespace members
//	@Description	Adds (or updates the admin flag of) users in the namespace
//	@Description	from the X-Namespace header. Global admin or admin of that
//	@Description	namespace.
//	@Accept			json
//	@Produce		json
//	@Param			X-Namespace	header		string					true	"namespace name or id"
//	@Param			request		body		PutNamespaceUsersInput	true	"members"
//	@Success		200			{object}	string
//	@Failure		400			{object}	string
//	@Failure		403			{object}	string
//	@Router			/api/admin/namespace/users [put]
func (self *Handlers) PutNamespaceUsers(
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

	var input PutNamespaceUsersInput
	if err := json.Unmarshal(b, &input); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Message:     "json unmarshal failed",
			OriginError: err,
		}
	}

	if err := self.namespaces.UpsertNamespaceMembers(
		ctx, namespaceId, input.UserIds, input.IsAdmin); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "failed to add namespace members",
			OriginError: err,
		}
	}

	return utils.Success(w)
}
