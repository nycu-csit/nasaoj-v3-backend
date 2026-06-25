package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type PutToolsInputTool struct {
	ToolId      int64                 `json:"toolId"`
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Testcase    models.Testcase       `json:"testcase"`
	Args        map[string]string     `json:"args"`
	InputArgs   []models.ToolInputArg `json:"inputArgs"`
	Tag         string                `json:"tag"`
	PreScript   models.Content        `json:"preScript"`
	PostScript  models.Content        `json:"postScript"`
	Cooldown    int64                 `json:"cooldown"`
}

type PutToolsOutputTool struct {
	SHA1   string `json:"sha1"`
	ToolId int64  `json:"toolId"`
	Name   string `json:"name"`
}

// PutTools
//
//	@Summary		Import tools
//	@Description	Imports or updates tool definitions. Admin only.
//	@Param			request	body	[]PutToolsInputTool	true	"tools"
//	@Accept			json
//	@Success		200	{object}	[]PutToolsOutputTool	"tools"
//	@Failure		401	{object}	string
//	@Router			/api/admin/tools [put]
func (self *Handlers) PutTools(
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

	var inputTools []PutToolsInputTool
	if err := json.Unmarshal(b, &inputTools); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Message:     "json unmarshal failed",
			OriginError: err,
		}
	}

	namespaceId := middlewares.NamespaceIdFromContext(ctx)

	var res []PutToolsOutputTool
	for _, input := range inputTools {
		tool := models.Tool{
			ToolId:      input.ToolId,
			NamespaceId: namespaceId,
			Name:        input.Name,
			Description: input.Description,
			Testcase:    input.Testcase,
			Args:        input.Args,
			InputArgs:   input.InputArgs,
			Tag:         input.Tag,
			PreScript:   input.PreScript,
			PostScript:  input.PostScript,
			Cooldown:    input.Cooldown,
		}
		sha1, err := self.tools.InsertTool(ctx, tool)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     fmt.Sprintf("insert tool '%s' failed", input.Name),
				OriginError: err,
			}
		}
		res = append(res, PutToolsOutputTool{
			SHA1:   sha1,
			ToolId: input.ToolId,
			Name:   input.Name,
		})
	}

	return bunrouter.JSON(w, res)
}
