package auth

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

type LoginPlainInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// PostLoginPlain
//
//	@Param		request	body	LoginPlainInput	true	"query params"
//	@Response	302		{}		string
//	@Router		/auth/login/plain [post]
func (self *Handlers) PostLoginPlain(
	w http.ResponseWriter, req bunrouter.Request,
) error {
	ctx := req.Context()

	b, err := io.ReadAll(req.Body)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Message:     "failed to read payload",
			OriginError: err,
		}
	}

	var input LoginPlainInput
	if err := json.Unmarshal(b, &input); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Message:     "failed to unmarshal json",
			OriginError: err,
		}
	}

	user, err := self.db.GetUserPass(ctx,
		input.Username, utils.HashPass(input.Password))
	if err != nil {
		if err == sql.ErrNoRows {
			return middlewares.HTTPError{
				StatusCode:  http.StatusUnauthorized,
				Message:     "failed to login, username / password incorrect",
				OriginError: err,
			}
		}

		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "failed to query database",
			OriginError: err,
		}
	}

	session := models.Session{
		Id:      user.Id,
		IsValid: true,
	}
	err = session.RotateRefreshToken()
	if err != nil && err != sql.ErrNoRows {
		return middlewares.HTTPError{
			StatusCode:  http.StatusUnauthorized,
			Message:     "failed to rotate refresh token",
			OriginError: err,
		}
	}

	if err := self.db.UpsertSession(ctx, &session); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "failed to upsert session",
			OriginError: err,
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    session.RefreshToken,
		Path:     "/",
		Secure:   false,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return utils.Redirect(w, "/")
}
