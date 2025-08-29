package api

import (
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type toolOutputTestcase struct {
	Name string `json:"name"`
}

type ToolOutput struct {
	ToolId     int64  `json:"toolId"`
	SHA1       string `json:"sha1"`
	ImportedAt int64  `json:"importedAt"`

	Name        string             `json:"name"`
	Description string             `json:"description"`
	Testcase    toolOutputTestcase `json:"testcase"`
	InputArgs   []string           `json:"inputArgs"`
	Tag         string             `json:"tag"`
	IsEnabled   bool               `json:"isEnabled"`
}

// GetAdminTools
//
//	@Produce	json
//	@Success	200	{object}	[]ToolOutput
//	@Failure	401
//	@Router		/api/admin/tools [get]
func getAdminToolsPlaceholder() {}

// GetTools
//
//	@Produce	json
//	@Success	200	{object}	[]ToolOutput
//	@Failure	401
//	@Router		/api/tools [get]
func GetTools(
	tools interfaces.ToolStorage,
	isAdmin bool,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		ctx := req.Context()

		var resTools map[string]models.Tool
		var err error
		if isAdmin {
			resTools, err = tools.GetAllTools(ctx)
		} else {
			resTools, err = tools.GetTools(ctx)
		}
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "failed to get tools",
				OriginError: err,
			}
		}

		ret := []ToolOutput{}
		for sha1, tool := range resTools {
			ret = append(ret, ToolOutput{
				Name:        tool.Name,
				SHA1:        sha1,
				ImportedAt:  tool.ImportedAt,
				ToolId:      tool.ToolId,
				Description: tool.Description,
				Testcase: toolOutputTestcase{
					Name: tool.Testcase.Name,
				},
				InputArgs: tool.InputArgs,
				Tag:       tool.Tag,
				IsEnabled: tool.IsEnabled,
			})
		}

		return bunrouter.JSON(w, ret)
	}
}
