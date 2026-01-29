package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type PostTokenInput struct {
	TokenType string `json:"tokenType"`
}

type PostTokenOutput struct {
	Token string `json:"token"`
}

// PostToken
//
//	@Param		request	body	PostTokenInput	true	"token type"
//	@Accept		json
//	@Success	200	{object}	PostTokenOutput
//	@Failure	400	{object}	string
//	@Failure	401	{object}	string
//
//	@Router		/api/admin/token [post]
func (self *Handlers) PostToken(
	w http.ResponseWriter, req bunrouter.Request,
) error {
	ctx := req.Context()

	b, err := io.ReadAll(req.Body)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Message:     "read payload failed",
			OriginError: err,
		}
	}

	var input PostTokenInput
	if err := json.Unmarshal(b, &input); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Message:     "json unmarshal failed",
			OriginError: err,
		}
	}

	var privs models.Privileges
	switch input.TokenType {
	case "worker":
		privs = models.NewWorkerPrivs()
	case "admin":
		privs = models.NewAdminPrivs("")
	default:
		return middlewares.HTTPError{
			StatusCode: http.StatusBadRequest,
			Message:    "token type not exist",
		}
	}

	token, err := self.tokens.GenerateTokenWithoutExpire(ctx, privs)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "token generate failed",
			OriginError: err,
		}
	}

	return bunrouter.JSON(w, PostTokenOutput{
		Token: token,
	})
}
