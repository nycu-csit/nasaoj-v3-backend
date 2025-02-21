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
func GetCallback(ctx context.Context, config oauth2.Config, verifier *oidc.IDTokenVerifier) func(http.ResponseWriter, bunrouter.Request) error {
	var secretKey = []byte(os.Getenv("JWT_SECRET"))

	return func(w http.ResponseWriter, req bunrouter.Request) error {
		fmt.Fprintf(os.Stderr, "GET /api/callback\n")

		oauth2token, err := config.Exchange(ctx, req.URL.Query().Get("code"))
		if err != nil {
			panic(err)
		}

		rawIDToken, ok := oauth2token.Extra("id_token").(string)
		if !ok {
			fmt.Fprintf(os.Stderr, "token missing")
			return nil
		}

		idToken, err := verifier.Verify(ctx, rawIDToken)
		if err != nil {
			panic(err)
		}

		var claim Claim
		if err := idToken.Claims(&claim); err != nil {
			panic(err)
		}

		jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256,
			models.JWTClaim{
				CsId:   claim.CsId,
				Groups: claim.Groups,
				Iat:    time.Now().Unix(),
				Exp:    time.Now().Add(time.Hour).Unix(),
			})

		jwtString, err := jwtToken.SignedString(secretKey)

		if err != nil {
			fmt.Printf("signing failed\n")
			return nil
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "jwt",
			Value:    jwtString,
			Secure:   false,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})

		w.Header().Set("Location", "/")
		w.WriteHeader(http.StatusFound)

		return nil
	}
}
