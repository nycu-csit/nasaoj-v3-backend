package api

import (
	"fmt"
	"net/http"
	"os"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
)

// GetTokens
//
//	@Produce	json
//	@Success	200	{object}	[]models.Token
//	@Failure	401	{object}	string
//	@Router		/api/admin/tokens [get]
func GetTokens(
	tokens interfaces.TokenStorage,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		fmt.Fprintf(os.Stderr, "GET /api/admin/tokens\n")
		ctx := req.Context()

		ret, err := tokens.GetTokens(ctx)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "get tokens failed",
				OriginError: err,
			}
		}
		return bunrouter.JSON(w, ret)
	}
}
