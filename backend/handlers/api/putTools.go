package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

// PutTools
//
//	@Param		request	body	map[string]models.Tool	true	"tool"
//	@Accept		json
//	@Success	200	{object}	string	"success"
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

	inputTools := make(map[string]models.Tool)
	if err := json.Unmarshal(b, &inputTools); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Message:     "json unmarshal failed",
			OriginError: err,
		}
	}

	for sha1, tool := range inputTools {
		if err := self.tools.InsertTool(ctx, tool); err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     fmt.Sprintf("insert tool '%s' failed", sha1),
				OriginError: err,
			}
		}
	}

	return utils.Success(w)
}
