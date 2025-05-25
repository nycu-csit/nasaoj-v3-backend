package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"slices"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/env"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type MeOutput struct {
	IsAdmin  bool                `json:"isAdmin"`
	JWTClaim models.JWTClaim     `json:"jwtClaim"`
	Resource models.UserResource `json:"resource"`
}

// GetMe
//
//	@Success	200	{object}	MeOutput
//	@Failure	401	{object}	string	"missing credential"
//	@Router		/api/me [get]
func GetMe(
	ctx context.Context,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		fmt.Fprintf(os.Stderr, "GET /api/me\n")

		jwtClaim, ok := req.Context().
			Value(middlewares.JWTKeyType("")).(models.JWTClaim)
		if !ok {
			return middlewares.HTTPError{
				StatusCode: http.StatusUnauthorized,
				Message:    "jwt claim missing",
			}
		}

		resource, ok := req.Context().
			Value(middlewares.UserResourceType("")).(models.UserResource)
		if !ok {
			return middlewares.HTTPError{
				StatusCode: http.StatusUnauthorized,
				Message:    "user resource missing",
			}
		}

		bunrouter.JSON(w, MeOutput{
			IsAdmin:  slices.Contains(jwtClaim.Groups, env.ADMIN_ROLE),
			JWTClaim: jwtClaim,
			Resource: resource,
		})
		return nil
	}
}
