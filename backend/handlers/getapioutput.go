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
			w.WriteHeader(http.StatusBadRequest)
			return fmt.Errorf("subId cannot convert to integer, %s\n", err)
		}
		stageName := req.URL.Query().Get("stage")
		testName := req.URL.Query().Get("testcase")

		resource, ok := req.Context().
			Value(middlewares.UserResourceType("")).(models.UserResource)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return fmt.Errorf("user resource missing\n")
		}

		submission := models.Submission{Id: subId}
		err = db.NewSelect().Model(&submission).WherePK().Scan(ctx)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return fmt.Errorf("submission id not found, %w\n", err)
		}

		if submission.CsId != resource.CsId {
			w.WriteHeader(http.StatusUnauthorized)
			return fmt.Errorf("submission owner not match")
		}

		fileName := fmt.Sprintf("%d-%s-%s-output", subId, stageName, testName)
		fmt.Fprintf(os.Stderr, "fileName: %s\n", fileName)

		output, err := mc.GetObject(ctx, MINIO_BUCKET, fileName,
			minio.GetObjectOptions{})
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return fmt.Errorf("minio get output failed, %s\n", err)
		}
		defer output.Close()

		b, err := io.ReadAll(output)

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "%s\n", string(b[:]))

		return nil
	}
}
