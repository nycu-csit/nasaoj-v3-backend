package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v5"
	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"golang.org/x/oauth2"
)

type ClaimProfile struct {
	Uid string `json:"uid"`
}

type Claim struct {
	CsId   string   `json:"csid"`
	Groups []string `json:"groups"`
}

// GetCallback
//
//	@Response	302	{}	string "302 Found"
//	@Router		/auth/callback [get]
func GetCallback(
	ctx context.Context,
	db interfaces.Database,
	config oauth2.Config,
	verifier *oidc.IDTokenVerifier,
) func(http.ResponseWriter, bunrouter.Request) error {
	var secretKey = []byte(os.Getenv("JWT_SECRET"))

	return func(w http.ResponseWriter, req bunrouter.Request) error {
		fmt.Fprintf(os.Stderr, "GET /api/callback\n")

		oauth2token, err := config.Exchange(ctx, req.URL.Query().Get("code"))
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusUnauthorized,
				Message:     "oauth code exchange failed",
				OriginError: err,
			}
		}

		rawIDToken, ok := oauth2token.Extra("id_token").(string)
		if !ok {
			return middlewares.HTTPError{
				StatusCode:  http.StatusUnauthorized,
				Message:     "oauth token missing",
				OriginError: err,
			}
		}

		idToken, err := verifier.Verify(ctx, rawIDToken)
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

		jwtClaim := models.JWTClaim{
			CsId:   claim.CsId,
			Groups: claim.Groups,
			Iat:    time.Now().Unix(),
			Exp:    time.Now().Add(time.Hour).Unix(),
		}
		jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaim)
		jwtString, err := jwtToken.SignedString(secretKey)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "jwt signing failed",
				OriginError: err,
			}
		}

		exist, err := db.ExistUserResource(ctx, jwtClaim.CsId)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusUnauthorized,
				Message:     "check exist failed",
				OriginError: err,
			}
		}
		if !exist {
			err := db.InsertNewUserFromResources(ctx, jwtClaim.CsId)
			if err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusInternalServerError,
					Message:     "user create failed",
					OriginError: err,
				}
			}
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "jwt",
			Value:    jwtString,
			Path:     "/",
			Secure:   false,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})

		w.Header().Set("Location", "/")
		w.WriteHeader(http.StatusFound)

		return nil
	}
}
