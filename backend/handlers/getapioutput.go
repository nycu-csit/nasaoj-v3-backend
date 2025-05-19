package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/minio/minio-go/v7"
	"github.com/uptrace/bun"
	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

// GetOutput
//
//	@Param		subId		query		number	true	"Submission id"
//	@Param		stage		query		string	true	"Stage name"
//	@Param		testcase	query		string	true	"Testcase name"
//	@Success	200			{string}	string	"Testcase result output"
//	@Failure	400			{string}	string
//	@Failure	404			{string}	string
//	@Router		/api/output [get]
func GetOutput(
	ctx context.Context,
	db *bun.DB,
	mc *minio.Client,
) func(http.ResponseWriter, bunrouter.Request) error {
	MINIO_BUCKET := os.Getenv("MINIO_BUCKET")

	return func(w http.ResponseWriter, req bunrouter.Request) error {
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

		resource, ok := req.Context().
			Value(middlewares.UserResourceType("")).(models.UserResource)
		if !ok {
			return middlewares.HTTPError{
				StatusCode: http.StatusUnauthorized,
				Message:    "user not exist",
			}
		}

		submission := models.Submission{Id: subId}
		err = db.NewSelect().Model(&submission).WherePK().Scan(ctx)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusNotFound,
				Message:     "submission id not found",
				OriginError: err,
			}
		}

		if submission.CsId != resource.CsId {
			return middlewares.HTTPError{
				StatusCode: http.StatusUnauthorized,
				Message:    "submission owner not match",
			}
		}

		fileName := fmt.Sprintf("%d-%s-%s-output", subId, stageName, testName)
		fmt.Fprintf(os.Stderr, "fileName: %s\n", fileName)

		output, err := mc.GetObject(ctx, MINIO_BUCKET, fileName,
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
