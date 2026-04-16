package auth

import (
	"database/sql"
	"net/http"
	"slices"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/spf13/viper"
	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/workflows"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"golang.org/x/oauth2"
)

type ClaimProfile struct {
	Uid string `json:"uid"`
}

type Claim struct {
	CsId   string   `json:"csid"`
	Groups []string `json:"groups"`
}

func oidcAdminStatus(groups []string) (isAdmin bool, sync bool) {
	adminRole := strings.TrimSpace(viper.GetString("admin-role"))
	if adminRole == "" {
		return false, false
	}
	return slices.Contains(groups, adminRole), true
}

// GetCallback
//
//	@Param			source	query	string	false	"Login source"	attribute(optional)
//	@Description	Oauth callback endpoint
//	@Response		302	{}	string	"302 Found"
//	@Router			/auth/callback [get]
func (self *Handlers) GetCallback(
	w http.ResponseWriter, req bunrouter.Request,
) error {
	ctx := req.Context()

	var err error
	var oauth2Token *oauth2.Token
	if req.URL.Query().Get("source") == "cli" {
		oauth2Token, err = self.cliOAuth2Config.Exchange(
			ctx, req.URL.Query().Get("code"))
	} else {
		oauth2Token, err = self.webOAuth2Config.Exchange(
			ctx, req.URL.Query().Get("code"))
	}
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusUnauthorized,
			Message:     "oauth code exchange failed",
			OriginError: err,
		}
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		return middlewares.HTTPError{
			StatusCode:  http.StatusUnauthorized,
			Message:     "oauth token missing",
			OriginError: err,
		}
	}

	var idToken *oidc.IDToken
	if req.URL.Query().Get("source") == "cli" {
		idToken, err = self.cliVerifier.Verify(ctx, rawIDToken)
	} else {
		idToken, err = self.webVerifier.Verify(ctx, rawIDToken)
	}
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusUnauthorized,
			Message:     "token verify failed",
			OriginError: err,
		}
	}

	var claim Claim
	if err := idToken.Claims(&claim); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusUnauthorized,
			Message:     "oauth claims missing",
			OriginError: err,
		}
	}

	// Check if user exists, register if not
	exist, err := self.db.ExistUserResourceBySourceId(ctx,
		models.LoginSourceCSSSO, claim.CsId)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusUnauthorized,
			Message:     "check user exist failed",
			OriginError: err,
		}
	}
	if !exist {
		// User not exist, register new user
		if !viper.GetBool("allow-user-register") {
			return middlewares.HTTPError{
				StatusCode: http.StatusUnauthorized,
				Message:    "user register is not allowed",
			}
		}

		err := workflows.RegisterUser(ctx, self.db, claim.CsId)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "user register failed",
				OriginError: err,
			}
		}
	}

	// Get user resource to get user ID and update IsAdmin
	user, err := self.db.GetUserResourceBySourceId(ctx,
		models.LoginSourceCSSSO, claim.CsId)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "get user failed",
			OriginError: err,
		}
	}

	// Update IsAdmin based on OIDC groups when role sync is enabled.
	isAdmin, shouldSyncAdmin := oidcAdminStatus(claim.Groups)
	if shouldSyncAdmin && user.IsAdmin != isAdmin {
		user.IsAdmin = isAdmin
		if err := self.db.UpsertUser(ctx, user); err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "update user admin status failed",
				OriginError: err,
			}
		}
	}

	// Create session with user ID (not CsId)
	session := models.Session{
		Id:      user.Id,
		IsValid: true,
	}
	err = session.RotateRefreshToken()
	if err != nil && err != sql.ErrNoRows {
		return middlewares.HTTPError{
			StatusCode:  http.StatusUnauthorized,
			Message:     "rotate refresh token failed",
			OriginError: err,
		}
	}

	if err := self.db.UpsertSession(ctx, &session); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "upsert session failed",
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

	if req.URL.Query().Get("source") == "cli" {
		return utils.Success(w)
	} else {
		return utils.Redirect(w, "/")
		cookie, err := req.Cookie("referer")
		if err == http.ErrNoCookie || (err == nil && cookie.Value == "") {
			return utils.Redirect(w, "/")
		} else {
			if err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusInternalServerError,
					Message:     "failed to get cookie",
					OriginError: err,
				}
			}
			return utils.Redirect(w, cookie.Value)
		}
	}
}
