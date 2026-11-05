package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type PutToolStateInput struct {
	ToolSHA1  string `json:"toolSHA1"`
	IsEnabled bool   `json:"isEnabled"`
}

// PutToolState
//
//	@Summary		Enable or disable a tool
//	@Description	Toggles the enabled state of a tool by SHA1 prefix. Admin only.
//	@Param			request	body	PutToolStateInput	true	"tool state"
//	@Accept			json
//	@Success		200	{object}	string	"success"
//	@Failure		400	{object}	string
//	@Failure		401	{object}	string
//	@Router			/api/admin/tool/state [put]
func (self *Handlers) PutToolState(
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

	var input PutToolStateInput
	if err := json.Unmarshal(b, &input); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Message:     "json unmarshal failed",
			OriginError: err,
		}
	}

	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("tool.sha1", input.ToolSHA1))

	if err = self.tools.UpdateToolState(ctx, input.ToolSHA1,
		middlewares.NamespaceIdsFromContext(ctx), input.IsEnabled); err != nil {
		// Enabling looks the tool up by prefix within the request's
		// namespaces; no match means it doesn't exist there.
		if errors.Is(err, sql.ErrNoRows) {
			return middlewares.HTTPError{
				StatusCode:  http.StatusNotFound,
				Message:     "tool not found in namespace",
				OriginError: err,
			}
		}
		return middlewares.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Message:     "tool put state failed",
			OriginError: err,
		}
	}

	return utils.Success(w)
}
