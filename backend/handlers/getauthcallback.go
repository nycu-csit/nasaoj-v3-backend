package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v5"
	"github.com/uptrace/bun"
	"github.com/uptrace/bunrouter"
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
	db *bun.DB,
	config oauth2.Config,
	verifier *oidc.IDTokenVerifier,
) func(http.ResponseWriter, bunrouter.Request) error {
	var secretKey = []byte(os.Getenv("JWT_SECRET"))

	return func(w http.ResponseWriter, req bunrouter.Request) error {
		fmt.Fprintf(os.Stderr, "GET /api/callback\n")

		oauth2token, err := config.Exchange(ctx, req.URL.Query().Get("code"))
		if err != nil {
			return fmt.Errorf("oauth code exchange failed, %s\n", err)
		}

		rawIDToken, ok := oauth2token.Extra("id_token").(string)
		if !ok {
			return fmt.Errorf("oauth token missing, %s\n", err)
		}

		idToken, err := verifier.Verify(ctx, rawIDToken)
		if err != nil {
			return fmt.Errorf("token verify failed, %s\n", err)
		}

		var claim Claim
		if err := idToken.Claims(&claim); err != nil {
			return fmt.Errorf("claims missing, %s\n", err)
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
			return fmt.Errorf("signing failed, %s\n", err)
		}

		exist, err := db.NewSelect().Model((*models.UserResource)(nil)).
			Where("csid = ?", jwtClaim.CsId).Exists(ctx)
		if err != nil {
			return fmt.Errorf("query UserResource failed, %s\n", err)
		}
		if !exist {
			var resource models.Resource
			err = db.NewDelete().Model((*models.Resource)(nil)).
				Where(
					"id IN (?)",
					db.NewSelect().Model((*models.Resource)(nil)).Column("id").Limit(1),
				).Returning("*").Scan(ctx, &resource)
			if err != nil {
				return fmt.Errorf("no resource left, err: %s\n", err)
			}

			user := resource.User(jwtClaim.CsId)
			_, err = db.NewInsert().Model(&user).Exec(ctx)
			if err != nil {
				return fmt.Errorf("user insert failed, err: %s\n", err)
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
