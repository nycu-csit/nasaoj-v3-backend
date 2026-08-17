package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type PostNamespaceInput struct {
	Name  string            `json:"name"`
	Title string            `json:"title"`
	Args  map[string]string `json:"args"`
}

// PostNamespace
//
//	@Summary		Create a namespace
//	@Description	Creates a namespace. Global admin only.
//	@Accept			json
//	@Produce		json
//	@Param			request	body		PostNamespaceInput	true	"namespace"
//	@Success		200		{object}	models.Namespace
//	@Failure		400		{object}	string
//	@Failure		401		{object}	string
//	@Router			/api/admin/namespace [post]
func (self *Handlers) PostNamespace(
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

	var input PostNamespaceInput
	if err := json.Unmarshal(b, &input); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Message:     "json unmarshal failed",
			OriginError: err,
		}
	}

	ns := models.Namespace{
		Name:      input.Name,
		Title:     input.Title,
		Args:      input.Args,
		IsEnabled: true,
	}
	if err := self.namespaces.CreateNamespace(ctx, &ns); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "failed to create namespace",
			OriginError: err,
		}
	}

	return bunrouter.JSON(w, ns)
}
