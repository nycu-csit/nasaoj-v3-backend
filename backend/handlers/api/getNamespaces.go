package api

import (
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
)

// GetNamespaces
//
//	@Summary		List namespaces
//	@Description	Lists namespaces. Global admin sees all; a namespace admin
//	@Description	sees only the namespaces they administer.
//	@Produce		json
//	@Success		200	{object}	[]models.Namespace
//	@Failure		401	{object}	string
//	@Router			/api/admin/namespaces [get]
func (self *Handlers) GetNamespaces(
	w http.ResponseWriter, req bunrouter.Request,
) error {
	ctx := req.Context()

	all, err := self.namespaces.GetNamespaces(ctx)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "failed to get namespaces",
			OriginError: err,
		}
	}

	privs, _ := ctx.Value(types.Privileges("")).(models.Privileges)
	if privs.CanAPIRead >= models.ScopeAll {
		return bunrouter.JSON(w, all)
	}

	// Namespace admin: filter to namespaces they administer.
	claim, ok := ctx.Value(types.AccessToken("")).(models.AccessTokenClaim)
	if !ok {
		return bunrouter.JSON(w, []models.Namespace{})
	}
	adminIn := make(map[int64]struct{}, len(claim.Resource.AdminIn))
	for _, id := range claim.Resource.AdminIn {
		adminIn[id] = struct{}{}
	}
	var visible []models.Namespace
	for _, ns := range all {
		if _, ok := adminIn[ns.Id]; ok {
			visible = append(visible, ns)
		}
	}
	return bunrouter.JSON(w, visible)
}
