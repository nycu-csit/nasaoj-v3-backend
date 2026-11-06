package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
)

type PostTokenInput struct {
	TokenType string `json:"tokenType"`

	// Description is required: it's how `judge admin get tokens` tells
	// tokens apart without decoding the secret itself.
	Description string `json:"description"`

	// UserId is required when TokenType is "user": the token's privileges
	// and namespace membership are resolved live from this user on every
	// request.
	UserId int64 `json:"userId,omitempty"`
}

type PostTokenOutput struct {
	Token string `json:"token"`
}

// PostToken
//
//	@Summary		Create an API token
//	@Description	Issues a new worker or admin API token. Admin only.
//	@Param			request	body	PostTokenInput	true	"token type"
//	@Accept			json
//	@Success		200	{object}	PostTokenOutput
//	@Failure		400	{object}	string
//	@Failure		401	{object}	string
//
//	@Router			/api/admin/token [post]
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

	input.Description = strings.TrimSpace(input.Description)
	if input.Description == "" {
		return middlewares.HTTPError{
			StatusCode: http.StatusBadRequest,
			Message:    "description required",
		}
	}

	var (
		privs  models.Privileges
		userId int64
	)
	switch input.TokenType {
	case "worker":
		privs = models.NewWorkerPrivs()
	case "admin":
		privs = models.NewAdminPrivs(0)
	case "user":
		if input.UserId == 0 {
			return middlewares.HTTPError{
				StatusCode: http.StatusBadRequest,
				Message:    "userId required for user token",
			}
		}
		user, err := self.db.GetUserResource(ctx, input.UserId)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusNotFound,
				Message:     "user not found",
				OriginError: err,
			}
		}
		userId = user.Id
		privs = userTokenPrivs(user)
	default:
		return middlewares.HTTPError{
			StatusCode: http.StatusBadRequest,
			Message:    "token type not exist",
		}
	}

	token, err := self.tokens.GenerateTokenWithoutExpire(
		ctx, privs, userId, input.Description)
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

// userTokenPrivs is the privileges stored on a user token. CheckAuth resolves
// them live from the user on every request anyway; this is what the token
// record shows.
func userTokenPrivs(user models.UserResource) models.Privileges {
	if user.IsAdmin {
		return models.NewAdminPrivs(user.Id)
	}
	return models.NewUserPrivs(user.Id)
}

type PostSelfTokenInput struct {
	// Description is required: it's how `judge admin get tokens` tells
	// tokens apart without decoding the secret itself.
	Description string `json:"description"`
}

// PostSelfToken
//
//	@Summary		Create an API token for yourself
//	@Description	Issues a user token for the caller. Its privileges and namespace membership are resolved live from the caller on every request, so it never grants more than the caller's own login.
//	@Param			request	body	PostSelfTokenInput	true	"token description"
//	@Accept			json
//	@Success		200	{object}	PostTokenOutput
//	@Failure		400	{object}	string
//	@Failure		401	{object}	string
//
//	@Router			/api/token [post]
func (self *Handlers) PostSelfToken(
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

	var input PostSelfTokenInput
	if err := json.Unmarshal(b, &input); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Message:     "json unmarshal failed",
			OriginError: err,
		}
	}

	input.Description = strings.TrimSpace(input.Description)
	if input.Description == "" {
		return middlewares.HTTPError{
			StatusCode: http.StatusBadRequest,
			Message:    "description required",
		}
	}

	// Static admin/worker tokens carry no owner: there is no user to issue
	// a token for.
	privs, _ := ctx.Value(types.Privileges("")).(models.Privileges)
	if privs.OwnerId == 0 {
		return middlewares.HTTPError{
			StatusCode: http.StatusBadRequest,
			Message:    "caller is not a user",
		}
	}

	user, err := self.db.GetUserResource(ctx, privs.OwnerId)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusNotFound,
			Message:     "user not found",
			OriginError: err,
		}
	}

	token, err := self.tokens.GenerateTokenWithoutExpire(
		ctx, userTokenPrivs(user), user.Id, input.Description)
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
