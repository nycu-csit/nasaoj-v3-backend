package api

import (
	"net/http"
	"slices"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
)

// GetTool
//
//	@Summary		Get a tool
//	@Description	Returns a single enabled tool by SHA1.
//	@Param			toolSHA1	query	string	true	"tool sha1"
//	@Produce		json
//	@Success		200	{object}	ToolOutput
//	@Failure		400	{object}	string
//	@Failure		401	{object}	string
//	@Router			/api/tool [get]
func (self *Handlers) GetTool(
	w http.ResponseWriter, req bunrouter.Request,
) error {
	ctx := req.Context()

	toolSHA1 := req.URL.Query().Get("toolSHA1")
	if toolSHA1 == "" {
		return middlewares.HTTPError{
			StatusCode: http.StatusBadRequest,
			Message:    "toolSHA1 empty",
		}
	}

	tool, err := self.tools.GetTool(ctx, toolSHA1)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "get tool failed",
			OriginError: err,
		}
	}

	if !slices.Contains(
		middlewares.NamespaceIdsFromContext(ctx), tool.NamespaceId) {
		return middlewares.HTTPError{
			StatusCode: http.StatusNotFound,
			Message:    "tool not in namespace",
		}
	}

	ret := ToolOutput{
		Name:        tool.Name,
		SHA1:        toolSHA1,
		ImportedAt:  tool.ImportedAt,
		ToolId:      tool.ToolId,
		Description: tool.Description,
		Testcase: toolOutputTestcase{
			Name:        tool.Testcase.Name,
			Timeout:     tool.Testcase.Timeout,
			GracePeriod: tool.Testcase.GracePeriod,
		},
		Tag:       tool.Tag,
		IsEnabled: tool.IsEnabled,
	}

	return bunrouter.JSON(w, ret)
}
