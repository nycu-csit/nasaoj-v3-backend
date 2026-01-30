package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"go.uber.org/zap"
)

// GetAdminStderr
//
//	@Param		subId		query		number	true	"Submission id"
//	@Param		stage		query		string	true	"Stage name"
//	@Param		testcase	query		string	true	"Testcase name"
//	@Success	200			{string}	string	"Testcase result stderr"
//	@Failure	400
//	@Failure	404
//	@Router		/api/admin/stderr [get]
func getAdminStderrPlaceholder() {}

// GetAdminStdout
//
//	@Param		subId		query		number	true	"Submission id"
//	@Param		stage		query		string	true	"Stage name"
//	@Param		testcase	query		string	true	"Testcase name"
//	@Success	200			{string}	string	"Testcase result stdout"
//	@Failure	400
//	@Failure	404
//	@Router		/api/admin/stdout [get]
func getAdminStdoutPlaceholder() {}

// GetStdout
//
//	@Param		subId		query		number	true	"Submission id"
//	@Param		stage		query		string	true	"Stage name"
//	@Param		testcase	query		string	true	"Testcase name"
//	@Success	200			{string}	string	"Testcase result stdout"
//	@Failure	400
//	@Failure	404
//	@Router		/api/stdout [get]
func (self *Handlers) GetOutput(
	target string,
	isAdmin bool,
) func(http.ResponseWriter, bunrouter.Request) error {

	return func(w http.ResponseWriter, req bunrouter.Request) error {
		ctx := req.Context()
		subId, err := strconv.ParseInt(req.URL.Query().Get("subId"), 10, 64)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     "subId cannot convert to integer",
				OriginError: err,
			}
		}
		stageName := req.URL.Query().Get("stage")
		testName := req.URL.Query().Get("testcase")

		submission, err := self.submissions.GetSubmission(ctx, subId)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusNotFound,
				Message:     "submission id not found",
				OriginError: err,
			}
		}

		if !isAdmin {
			privs, ok := req.Context().
				Value(types.Privileges("")).(models.Privileges)
			if !ok {
				return middlewares.HTTPError{
					StatusCode: http.StatusUnauthorized,
					Message:    "user not exist",
				}
			}

			if submission.OwnerId != privs.OwnerId {
				return middlewares.HTTPError{
					StatusCode: http.StatusForbidden,
					Message:    "submission owner not match",
				}
			}
		}

		fileName := fmt.Sprintf("%d-%s-%s-%s", subId, stageName, testName, target)
		tracing.Logger.Ctx(ctx).
			Info("get output from minio",
				zap.String("file.name", fileName))

		output, err := self.s3.GetFile(ctx, fileName)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusNotFound,
				Message:     "get output failed",
				OriginError: err,
			}
		}

		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusNotFound,
				Message:     "get output failed",
				OriginError: err,
			}
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, err = fmt.Fprintf(w, "%s\n", string(output))
		return err
	}
}
