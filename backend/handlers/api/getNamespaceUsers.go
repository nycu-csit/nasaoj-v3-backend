package api

import (
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"go.uber.org/zap"
)

// GetNamespaceUsers
//
//	@Summary		List namespace members
//	@Description	Lists members of the namespace from the X-Namespace header.
//	@Description	Global admin or admin of that namespace.
//	@Produce		json
//	@Param			X-Namespace	header		string	true	"namespace name or id"
//	@Success		200			{object}	[]models.UserNamespace
//	@Failure		403			{object}	string
//	@Router			/api/admin/namespace/users [get]
func (self *Handlers) GetNamespaceUsers(
	w http.ResponseWriter, req bunrouter.Request,
) error {
	ctx := req.Context()
	namespaceId := middlewares.NamespaceIdFromContext(ctx)

	members, err := self.namespaces.GetNamespaceMembers(ctx, namespaceId)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to get namespace members",
				zap.Error(err))
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "failed to get namespace members",
			OriginError: err,
		}
	}

	return bunrouter.JSON(w, members)
}
