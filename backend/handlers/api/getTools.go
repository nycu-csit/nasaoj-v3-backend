package api

import (
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type toolOutputTestcase struct {
	Name        string `json:"name"`
	Timeout     int64  `json:"timeout"`
	GracePeriod int64  `json:"gracePeriod"`
}

type ToolOutput struct {
	ToolId     int64  `json:"toolId"`
	SHA1       string `json:"sha1"`
	ImportedAt int64  `json:"importedAt"`

	Name        string                `json:"name"`
	Description string                `json:"description"`
	Testcase    toolOutputTestcase    `json:"testcase"`
	InputArgs   []models.ToolInputArg `json:"inputArgs"`
	Tag         string                `json:"tag"`
	IsEnabled   bool                  `json:"isEnabled"`
}

// GetAdminTools
//
//	@Summary		List all tools (admin)
//	@Description	Returns all tools, including disabled ones. Admin only.
//	@Produce		json
//	@Success		200	{object}	[]ToolOutput
//	@Failure		401
//	@Router			/api/admin/tools [get]
func getAdminToolsPlaceholder() {}

// GetTools
//
//	@Summary		List tools
//	@Description	Returns the enabled tools visible to the caller.
//	@Produce		json
//	@Success		200	{object}	[]ToolOutput
//	@Failure		401
//	@Router			/api/tools [get]
func (self *Handlers) GetTools(
	isAdmin bool,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		ctx := req.Context()

		// Scope the listing to the requested namespace(s) in storage, so other
		// namespaces' tools are never loaded. Storage reads nil as "every
		// namespace", so fail closed if none were resolved.
		namespaceIds := middlewares.NamespaceIdsFromContext(ctx)
		if len(namespaceIds) == 0 {
			return bunrouter.JSON(w, []ToolOutput{})
		}

		var resTools map[string]models.Tool
		var err error
		if isAdmin {
			resTools, err = self.tools.GetAllTools(ctx, namespaceIds)
		} else {
			resTools, err = self.tools.GetTools(ctx, namespaceIds)
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
					Name:        tool.Testcase.Name,
					Timeout:     tool.Testcase.Timeout,
					GracePeriod: tool.Testcase.GracePeriod,
				},
				InputArgs: tool.InputArgs,
				Tag:       tool.Tag,
				IsEnabled: tool.IsEnabled,
			})
		}

		return bunrouter.JSON(w, ret)
	}
}
