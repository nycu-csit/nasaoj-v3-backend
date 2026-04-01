package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

type PutUsersPlainInputUser struct {
	Username string            `json:"username" yaml:"username"`
	Password string            `json:"password" yaml:"password"`
	Args     map[string]string `json:"args" yaml:"args"`
	IsAdmin  bool              `json:"isAdmin" yaml:"isAdmin"`
}

// PutUsersPlain
//
//	@Param		request	body	[]PutUsersPlainInputUser	true	"request body"
//	@Accept		json
//	@Success	200
//	@Failure	401
//	@Router		/api/admin/users/plain [put]
func (self *Handlers) PutUsersPlain(
	w http.ResponseWriter, req bunrouter.Request,
) error {
	ctx := req.Context()

	b, err := io.ReadAll(req.Body)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Message:     "failed to read payload",
			OriginError: err,
		}
	}

	var inputUsers []PutUsersPlainInputUser
	if err := json.Unmarshal(b, &inputUsers); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Message:     "failed to unmarshal json",
			OriginError: err,
		}
	}

	var users []models.UserResource
	var creds []models.PlainCred
	for _, user := range inputUsers {
		users = append(users, models.UserResource{
			Username: user.Username,
			Args:     user.Args,
			IsAdmin:  user.IsAdmin,
			Source:   models.LoginSourcePlain,
		})
		creds = append(creds, models.PlainCred{
			Passhash: utils.HashPass(user.Password),
		})
	}

	if err := self.db.InsertUserResources(ctx, users, creds); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "failed to upsert users",
			OriginError: err,
		}
	}

	return utils.Success(w)
}
