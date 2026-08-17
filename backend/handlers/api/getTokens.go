package api

import (
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

// GetTokensOutput is kept as an alias to models.Token: the token itself now
// carries its own SHA256 field (it's a real Postgres row, not a Redis hash
// entry keyed by SHA256), so no separate wrapper is needed.
type GetTokensOutput = models.Token

// GetTokens
//
//	@Summary		List API tokens
//	@Description	Returns all issued API tokens and their privileges. Admin only.
//	@Produce		json
//	@Success		200	{object}	[]GetTokensOutput
//	@Failure		401	{object}	string
//	@Router			/api/admin/tokens [get]
func (self *Handlers) GetTokens(
	w http.ResponseWriter, req bunrouter.Request,
) error {
	ctx := req.Context()

	res, err := self.tokens.GetTokens(ctx)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "get tokens failed",
			OriginError: err,
		}
	}

	return bunrouter.JSON(w, res)
}
