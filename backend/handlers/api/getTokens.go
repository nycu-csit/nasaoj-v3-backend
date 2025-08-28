package api

import (
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type GetTokensOutput struct {
	SHA256 string `json:"sha256"`

	models.Token `json:",inline"`
}

// GetTokens
//
//	@Produce	json
//	@Success	200	{object}	[]GetTokensOutput
//	@Failure	401	{object}	string
//	@Router		/api/admin/tokens [get]
func GetTokens(
	tokens interfaces.TokenStorage,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		ctx := req.Context()

		res, err := tokens.GetTokens(ctx)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "get tokens failed",
				OriginError: err,
			}
		}

		ret := []GetTokensOutput{}
		for key, token := range res {
			ret = append(ret, GetTokensOutput{
				SHA256: key,
				Token:  token,
			})
		}

		return bunrouter.JSON(w, ret)
	}
}
