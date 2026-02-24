package auth

import (
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

// PutLogout
//
//	@Description	Logout endpoint
//	@Response		302	{}	string	"302 Found"
//	@Router			/auth/logout [put]
func (self *Handlers) PutLogout(
	w http.ResponseWriter, req bunrouter.Request,
) error {
	ctx := req.Context()

	refreshTokenClaim, ok := req.Context().
		Value(types.RefreshToken("")).(models.RefreshTokenClaim)
	if !ok {
		return middlewares.HTTPError{
			StatusCode: http.StatusBadRequest,
			Message:    "refresh token not exist",
		}
	}
	if err := self.db.InvalidateSession(ctx, refreshTokenClaim.Id); err != nil {
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

	return utils.Success(w)
}
