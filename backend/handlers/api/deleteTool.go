package api

import (
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// DeleteTool
//
//	@Summary		Delete a tool
//	@Description	Deletes a tool and its submissions by SHA1 prefix. Admin only.
//	@Param			toolSHA1	query		string	true	"Tool SHA1"
//	@Success		200			{object}	string
//	@Failure		400			{object}	string
//	@Failure		401			{object}	string
//	@Router			/api/admin/tool [delete]
func (self *Handlers) DeleteTool(
	w http.ResponseWriter, req bunrouter.Request,
) error {
	ctx := req.Context()

	toolSHA1 := req.URL.Query().Get("toolSHA1")
	if toolSHA1 == "" {
		tracing.Logger.Ctx(ctx).
			Warn("tool sha1 not exist")
		return middlewares.HTTPError{
			StatusCode: http.StatusBadRequest,
			Message:    "tool sha1 not provided",
		}
	}

	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("tool.sha1", toolSHA1))

	if err := self.tools.DeleteToolAndSubmissions(ctx, toolSHA1); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "failed to delete tool and submissions",
			OriginError: err,
		}
	}
	return utils.Success(w)
}
