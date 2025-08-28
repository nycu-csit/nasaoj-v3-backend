package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

type PutTokenStateInput struct {
	Token string `json:"token"`
	State bool   `json:"state"`
}

// PutTokenState
//
//	@Accept		json
//	@Param		request	body		PutTokenStateInput	true	"query params"
//	@Success	200		{object}	string
//	@Failure	400		{object}	string
//	@Failure	401		{object}	string
//	@Router		/api/admin/token-state [put]
func PutTokenState(
	tokens interfaces.TokenStorage,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		ctx := req.Context()

		b, err := io.ReadAll(req.Body)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     "read payload failed",
				OriginError: err,
			}
		}

		var input PutTokenStateInput
		if err := json.Unmarshal(b, &input); err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     "json unmarshal failed",
				OriginError: err,
			}
		}

		if input.State {
			if err := tokens.EnableToken(ctx, input.Token); err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusInternalServerError,
					Message:     "enable token failed",
					OriginError: err,
				}
			}
		} else {
			if err := tokens.DisableToken(ctx, input.Token); err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusInternalServerError,
					Message:     "disable token failed",
					OriginError: err,
				}
			}
		}

		return utils.Success(w)
	}
}
