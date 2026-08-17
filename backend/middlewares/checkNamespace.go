package middlewares

import (
	"context"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/spf13/viper"
	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
)

// NamespaceIdsFromContext returns the set of namespaces a request operates on
// (resolved and authorized by CheckNamespace).
func NamespaceIdsFromContext(ctx context.Context) []int64 {
	ids, _ := ctx.Value(types.NamespaceId("")).([]int64)
	return ids
}

// NamespaceIdFromContext returns the primary (first) namespace, for endpoints
// that target a single namespace (import, submit, scores).
func NamespaceIdFromContext(ctx context.Context) int64 {
	ids := NamespaceIdsFromContext(ctx)
	if len(ids) == 0 {
		return 0
	}
	return ids[0]
}

func resolveNamespaceRef(
	ctx context.Context,
	namespaces interfaces.NamespaceStorage,
	ref string,
) (models.Namespace, error) {
	ref = strings.TrimSpace(ref)
	if id, err := strconv.ParseInt(ref, 10, 64); err == nil {
		return namespaces.GetNamespace(ctx, id)
	}
	return namespaces.GetNamespaceByName(ctx, ref)
}

// CheckNamespace resolves the request's target namespace(s) and authorizes the
// caller for the requested access level. Namespaces come from one of several
// sources:
//
//   - the X-Namespace header (a comma-separated list of names/ids), or
//   - the caller's JWT membership, when X-Namespace is "*"/"all" or the
//     configured namespace-source is "jwt", or
//   - the default namespace, when nothing is specified.
//
// A global admin (CanAPI* == ScopeAll) bypasses membership; for the JWT source a
// global admin resolves to all enabled namespaces. The resolved, authorized set
// is stored in the context.
func CheckNamespace(
	namespaces interfaces.NamespaceStorage,
	level models.NamespaceAccess,
) func(bunrouter.HandlerFunc) bunrouter.HandlerFunc {
	return func(next bunrouter.HandlerFunc) bunrouter.HandlerFunc {
		return func(w http.ResponseWriter, req bunrouter.Request) error {
			ctx := req.Context()

			privs, _ := ctx.Value(types.Privileges("")).(models.Privileges)
			globalRead := privs.CanAPIRead >= models.ScopeAll
			globalAdmin := privs.CanAPIWrite >= models.ScopeAll
			globalBypass := globalAdmin ||
				(level == models.NamespaceRead && globalRead)

			claim, hasClaim := ctx.Value(
				types.AccessToken("")).(models.AccessTokenClaim)

			header := strings.TrimSpace(req.Header.Get("X-Namespace"))
			fromJWT := header == "*" || header == "all" ||
				(header == "" && viper.GetString("namespace-source") == "jwt")

			var ids []int64

			switch {
			case fromJWT:
				// Derive the namespaces from the caller's identity.
				if globalBypass {
					all, err := namespaces.GetNamespaces(ctx)
					if err != nil {
						return HTTPError{
							StatusCode:  http.StatusInternalServerError,
							Message:     "failed to list namespaces",
							OriginError: err,
						}
					}
					for _, ns := range all {
						if ns.IsEnabled {
							ids = append(ids, ns.Id)
						}
					}
				} else if hasClaim {
					if level == models.NamespaceAdmin {
						ids = append(ids, claim.Resource.AdminIn...)
					} else {
						ids = append(ids, claim.Resource.Namespaces...)
					}
				}
				if len(ids) == 0 {
					return HTTPError{
						StatusCode: http.StatusForbidden,
						Message:    "no accessible namespaces",
					}
				}

			case header != "":
				for _, ref := range strings.Split(header, ",") {
					ns, err := resolveNamespaceRef(ctx, namespaces, ref)
					if err != nil {
						return HTTPError{
							StatusCode:  http.StatusNotFound,
							Message:     "namespace not found",
							OriginError: err,
						}
					}
					if !ns.IsEnabled && !globalAdmin {
						return HTTPError{
							StatusCode: http.StatusForbidden,
							Message:    "namespace disabled",
						}
					}
					if !authorizeNamespace(globalBypass, claim, hasClaim, level, ns.Id) {
						return HTTPError{
							StatusCode: http.StatusForbidden,
							Message:    "insufficient namespace access",
						}
					}
					ids = append(ids, ns.Id)
				}

			default:
				ns, err := namespaces.GetNamespace(ctx, models.DefaultNamespaceId)
				if err != nil {
					return HTTPError{
						StatusCode:  http.StatusNotFound,
						Message:     "default namespace not found",
						OriginError: err,
					}
				}
				if !ns.IsEnabled && !globalAdmin {
					return HTTPError{
						StatusCode: http.StatusForbidden,
						Message:    "namespace disabled",
					}
				}
				if !authorizeNamespace(globalBypass, claim, hasClaim, level, ns.Id) {
					return HTTPError{
						StatusCode: http.StatusForbidden,
						Message:    "insufficient namespace access",
					}
				}
				ids = []int64{ns.Id}
			}

			ctx = context.WithValue(ctx, types.NamespaceId(""), ids)
			return next(w, req.WithContext(ctx))
		}
	}
}

func authorizeNamespace(
	globalBypass bool,
	claim models.AccessTokenClaim,
	hasClaim bool,
	level models.NamespaceAccess,
	id int64,
) bool {
	if globalBypass {
		return true
	}
	if !hasClaim {
		return false
	}
	if level == models.NamespaceAdmin {
		return slices.Contains(claim.Resource.AdminIn, id)
	}
	return slices.Contains(claim.Resource.Namespaces, id)
}
