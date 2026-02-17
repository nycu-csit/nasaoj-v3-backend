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

type PutUsersCSSSOInput struct {
	CsId string            `json:"csId" yaml:"csId"`
	Args map[string]string `json:"args" yaml:"args`
}

// PutUsersCSSSO
//
//	@Param		request	body	[]PutUsersCSSSOInput	true	"request body"
//	@Accept		json
//	@Success	200	{object}	string	"success"
//	@Failure	401	{object}	string
//	@Router		/api/admin/users/cssso [put]
func (self *Handlers) PutUsersCSSSO(
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

	var inputUsers []PutUsersCSSSOInput
	if err := json.Unmarshal(b, &inputUsers); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Message:     "json unmarshal failed",
			OriginError: err,
		}
	}

	var users []models.UserResource
	for _, user := range inputUsers {
		users = append(users, models.UserResource{
			Username: user.CsId,
			Args:     user.Args,
			Source:   models.LoginSourceCSSSO,
			SourceId: user.CsId,
		})
	}

	if err := self.db.InsertUserResources(ctx, users, nil); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "insert user resources failed",
			OriginError: err,
		}
	}

	return utils.Success(w)
}
