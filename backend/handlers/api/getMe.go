package api

import (
	"net/http"
	"slices"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
)

// MeNamespace is a namespace the caller can access. Namespace args are left
// out on purpose: they are admin-managed job defaults, not member-facing.
type MeNamespace struct {
	Id        int64  `json:"id"`
	Name      string `json:"name"`
	Title     string `json:"title"`
	IsEnabled bool   `json:"isEnabled"`
	// IsAdmin is whether the caller administers this namespace, either as its
	// namespace admin or as a global admin.
	IsAdmin bool `json:"isAdmin"`
}

type MeOutput struct {
	IsAdmin          bool                    `json:"isAdmin"`
	AccessTokenClaim models.AccessTokenClaim `json:"accessToken"`
	// Namespaces are what the caller can access right now: every namespace
	// for a global admin, otherwise the access token's membership.
	Namespaces []MeNamespace `json:"namespaces"`
}

// GetMe
//
//	@Summary		Get the current user
//	@Description	Returns the authenticated user's profile and resource args,
//	@Description	and the namespaces they can access (all of them for a global
//	@Description	admin).
//	@Success		200	{object}	MeOutput
//	@Failure		401	{object}	string	"missing credential"
//	@Router			/api/me [get]
func (self *Handlers) GetMe(
	w http.ResponseWriter, req bunrouter.Request,
) error {
	ctx := req.Context()

	claim, ok := ctx.Value(types.AccessToken("")).(models.AccessTokenClaim)
	if !ok {
		return middlewares.HTTPError{
			StatusCode: http.StatusUnauthorized,
			Message:    "jwt claim missing",
		}
	}

	all, err := self.namespaces.GetNamespaces(ctx)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "failed to get namespaces",
			OriginError: err,
		}
	}

	// A global admin bypasses membership in CheckNamespace, so it can access
	// and administer every namespace.
	isGlobalAdmin := claim.Resource.IsAdmin

	namespaces := []MeNamespace{}
	for _, ns := range all {
		if !isGlobalAdmin && !slices.Contains(claim.Resource.Namespaces, ns.Id) {
			continue
		}
		namespaces = append(namespaces, MeNamespace{
			Id:        ns.Id,
			Name:      ns.Name,
			Title:     ns.Title,
			IsEnabled: ns.IsEnabled,
			IsAdmin: isGlobalAdmin ||
				slices.Contains(claim.Resource.AdminIn, ns.Id),
		})
	}

	return bunrouter.JSON(w, MeOutput{
		IsAdmin:          isGlobalAdmin,
		AccessTokenClaim: claim,
		Namespaces:       namespaces,
	})
}
