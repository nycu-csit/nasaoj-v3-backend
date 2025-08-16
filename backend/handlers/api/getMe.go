package api

import (
	"fmt"
	"net/http"
	"os"
	"slices"

	"github.com/spf13/viper"
	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
)

type MeOutput struct {
	IsAdmin          bool                    `json:"isAdmin"`
	AccessTokenClaim models.AccessTokenClaim `json:"accessToken"`
}

// GetMe
//
//	@Success	200	{object}	MeOutput
//	@Failure	401	{object}	string	"missing credential"
//	@Router		/api/me [get]
func GetMe(w http.ResponseWriter, req bunrouter.Request) error {
	fmt.Fprintf(os.Stderr, "GET /api/me\n")

	// fmt.Fprintf(os.Stderr, "[INFO] Context: %v\n", req.Context())
	// fmt.Fprintf(os.Stderr, "[INFO] Claim: %v\n", req.Context().Value(types.AccessToken("")))
	claim, ok := req.Context().
		Value(types.AccessToken("")).(models.AccessTokenClaim)
	if !ok {
		return middlewares.HTTPError{
			StatusCode: http.StatusUnauthorized,
			Message:    "jwt claim missing",
		}
	}

	bunrouter.JSON(w, MeOutput{
		IsAdmin: slices.Contains(
			claim.Groups, viper.GetString("admin-role")),
		AccessTokenClaim: claim,
	})
	return nil
}
