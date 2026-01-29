package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

// PutUser
//
//	@Param		request	body	map[string]string	true	"User resource"
//	@Accept		json
//	@Success	200	{object}	string	"success"
//	@Failure	401	{object}	string
//	@Router		/api/user [put]
func (self *Handlers) PutUser(
	w http.ResponseWriter, req bunrouter.Request,
) error {
	ctx := req.Context()

	privs, ok := ctx.Value(types.Privileges("")).(models.Privileges)
	if !ok {
		return middlewares.HTTPError{
			StatusCode: http.StatusUnauthorized,
			Message:    "privs not exist",
		}
	}

	b, err := io.ReadAll(req.Body)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Message:     "read payload failed",
			OriginError: err,
		}
	}

	var inputArgs map[string]string
	if err := json.Unmarshal(b, &inputArgs); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Message:     "json unmarshal failed",
			OriginError: err,
		}
	}

	if err := self.db.InsertUserResources(ctx,
		[]models.UserResource{models.UserResource{
			CsId: privs.OwnerCsId,
			Args: inputArgs,
		}}); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "insert user resources failed",
			OriginError: err,
		}
	}

	return utils.Success(w)
}
