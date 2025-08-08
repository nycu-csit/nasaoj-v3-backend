package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/minio/minio-go/v7"
	"github.com/spf13/viper"
	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

// GetAdminStderr
//
//	@Param		subId		query		number	true	"Submission id"
//	@Param		stage		query		string	true	"Stage name"
//	@Param		testcase	query		string	true	"Testcase name"
//	@Success	200			{string}	string	"Testcase result stderr"
//	@Failure	400			{string}	string
//	@Failure	404			{string}	string
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
//	@Failure	400			{string}	string
//	@Failure	404			{string}	string
//	@Router		/api/stdout [get]
func GetOutput(
	db interfaces.Database,
	mc *minio.Client,
	target string,
	isAdmin bool,
) func(http.ResponseWriter, bunrouter.Request) error {

	return func(w http.ResponseWriter, req bunrouter.Request) error {
		ctx := req.Context()
		fmt.Fprintf(os.Stderr, "GET /api/output\n")

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

		submission, err := db.GetSubmission(ctx, subId)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusNotFound,
				Message:     "submission id not found",
				OriginError: err,
			}
		}

		if !isAdmin {
			claim, ok := req.Context().
				Value(middlewares.AccessTokenType("")).(models.AccessTokenClaim)
			if !ok {
				return middlewares.HTTPError{
					StatusCode: http.StatusUnauthorized,
					Message:    "user not exist",
				}
			}

			if submission.CsId != claim.CsId {
				return middlewares.HTTPError{
					StatusCode: http.StatusUnauthorized,
					Message:    "submission owner not match",
				}
			}
		}

		fileName := fmt.Sprintf("%d-%s-%s-%s", subId, stageName, testName, target)
		fmt.Fprintf(os.Stderr, "fileName: %s\n", fileName)

		output, err := mc.GetObject(ctx, viper.GetString("minio-bucket"), fileName,
			minio.GetObjectOptions{})
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusNotFound,
				Message:     "get output failed",
				OriginError: err,
			}
		}
		defer output.Close()

		b, err := io.ReadAll(output)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusNotFound,
				Message:     "get output failed",
				OriginError: err,
			}
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "%s\n", string(b[:]))

		return nil
	}
}
