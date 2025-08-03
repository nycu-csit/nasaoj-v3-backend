package auth

import (
	"fmt"
	"net/http"
	"os"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

// PutLogout
//
//	@Description	Logout endpoint
//	@Response		302	{}	string	"302 Found"
//	@Router			/auth/logout [put]
func PutLogout(
	db interfaces.Database,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		ctx := req.Context()
		fmt.Fprintf(os.Stderr, "PUT /api/logout\n")

		refreshTokenClaim, ok := req.Context().
			Value(middlewares.RefreshTokenType("")).(models.RefreshTokenClaim)
		if !ok {
			return middlewares.HTTPError{
				StatusCode: http.StatusBadRequest,
				Message:    "refresh token not exist",
			}
		}
		if err := db.InvalidateSession(ctx, refreshTokenClaim.CsId); err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "invalidate session failed",
				OriginError: err,
			}
		}

		http.SetCookie(w, &http.Cookie{
			Name:   "access_token",
			MaxAge: -1,
		})
		http.SetCookie(w, &http.Cookie{
			Name:   "refresh_token",
			MaxAge: -1,
		})

		w.Header().Set("Location", "/")
		w.WriteHeader(http.StatusFound)
		return nil
	}
}
