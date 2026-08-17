package api

import (
	"net/http"
	"slices"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
)

// GetAdminProblem
//
//	@Summary		Get a problem (admin)
//	@Description	Returns a single problem by SHA1, including disabled ones.
//	@Description	Pass detail=true for the full problem definition (every
//	@Description	field, including testcase file hashes) instead of the
//	@Description	summary. Admin only.
//	@Param			problemSHA1	query	string	true	"problem sha1"
//	@Param			detail		query	bool	false	"return the full problem definition"
//	@Produce		json
//	@Success		200	{object}	ProblemOutput
//	@Failure		400	{object}	string
//	@Failure		401	{object}	string
//	@Router			/api/admin/problem [get]
func getAdminProblemPlaceholder() {}

// GetProblem
//
//	@Summary		Get a problem
//	@Description	Returns a single enabled problem by SHA1.
//	@Param			problemSHA1	query	string	true	"problem sha1"
//	@Produce		json
//	@Success		200	{object}	ProblemOutput
//	@Failure		400	{object}	string
//	@Failure		401	{object}	string
//	@Router			/api/problem [get]
func (self *Handlers) GetProblem(
	isAdmin bool,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		ctx := req.Context()

		problemSHA1 := req.URL.Query().Get("problemSHA1")
		if problemSHA1 == "" {
			return middlewares.HTTPError{
				StatusCode: http.StatusBadRequest,
				Message:    "problemSHA1 empty",
			}
		}

		problem, err := self.problems.GetProblem(ctx, problemSHA1)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "get problem failed",
				OriginError: err,
			}
		}

		if !slices.Contains(
			middlewares.NamespaceIdsFromContext(ctx), problem.NamespaceId) {
			return middlewares.HTTPError{
				StatusCode: http.StatusNotFound,
				Message:    "problem not in namespace",
			}
		}

		if isAdmin && req.URL.Query().Get("detail") == "true" {
			return bunrouter.JSON(w, ProblemDetailOutput{
				Problem: problem,
				SHA1:    problemSHA1,
			})
		}

		return bunrouter.JSON(w, toProblemOutput(problemSHA1, problem))
	}
}
