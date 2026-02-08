package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type PutToolsInputTool struct {
	SHA1 string `json:"sha1"`

	ToolId      int64             `json:"toolId"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Testcase    models.Testcase   `json:"testcase"`
	Args        map[string]string `json:"args"`
	InputArgs   []string          `json:"inputArgs"`
	Tag         string            `json:"tag"`
	PreScript   models.Content    `json:"preScript"`
	PostScript  models.Content    `json:"postScript"`
	Cooldown    int64             `json:"cooldown"`
}

// PutTools
//
//	@Param		request	body	[]PutToolsInputTool	true	"tool"
//	@Accept		json
//	@Success	200	{object}	[]string	"SHA1s"
//	@Failure	401	{object}	string
//	@Router		/api/admin/tools [put]
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

	var res strings.Builder
	for _, input := range inputTools {
		tool := models.Tool{
			ToolId:      input.ToolId,
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
		if err := self.tools.InsertTool(ctx, tool); err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     fmt.Sprintf("insert tool '%s' failed", input.SHA1),
				OriginError: err,
			}
		}
		io.WriteString(&res, input.SHA1+"\n")
	}

	_, err = io.WriteString(w, res.String())
	return err
}
