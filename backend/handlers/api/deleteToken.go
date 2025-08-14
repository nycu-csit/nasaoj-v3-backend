package api

import (
	"fmt"
	"net/http"
	"os"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
)

// DeleteToken
//
//	@Param		token	query		string	true	"Token"	attribute(optional)
//	@Success	200		{object}	string
//	@Failure	400		{object}	string
//	@Failure	401		{object}	string
//	@Router		/api/admin/token [delete]
func DeleteToken(
	tokens interfaces.TokenStorage,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		ctx := req.Context()
		fmt.Fprintf(os.Stderr, "DELETE /api/admin/token\n")

		token := req.URL.Query().Get("token")
		if err := tokens.DisableToken(ctx, token); err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "token disable failed",
				OriginError: err,
			}
		}

		fmt.Fprintf(w, "success")
		return nil
	}
}
