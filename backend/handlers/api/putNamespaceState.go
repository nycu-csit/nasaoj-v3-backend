package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

type PutNamespaceStateInput struct {
	Id        int64 `json:"id"`
	IsEnabled bool  `json:"isEnabled"`
}

// PutNamespaceState
//
//	@Summary		Enable or disable a namespace
//	@Description	Toggles a namespace's enabled state. Global admin only.
//	@Accept			json
//	@Produce		json
//	@Param			request	body		PutNamespaceStateInput	true	"state"
//	@Success		200		{object}	string
//	@Failure		400		{object}	string
//	@Router			/api/admin/namespace/state [put]
func (self *Handlers) PutNamespaceState(
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

	var input PutNamespaceStateInput
	if err := json.Unmarshal(b, &input); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Message:     "json unmarshal failed",
			OriginError: err,
		}
	}

	if err := self.namespaces.UpdateNamespaceState(
		ctx, input.Id, input.IsEnabled); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "failed to update namespace state",
			OriginError: err,
		}
	}

	return utils.Success(w)
}
