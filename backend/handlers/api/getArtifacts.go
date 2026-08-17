package api

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// GetAdminArtifacts
//
//	@Summary		Get a job's kept artifacts (admin)
//	@Description	Returns the tar of files a testcase declared as artifacts,
//	@Description	for any submission. Admin only.
//	@Param			subId		query	number	true	"Submission id"
//	@Param			stage		query	string	true	"Stage name"
//	@Param			testcase	query	string	true	"Testcase name"
//	@Success		200			{file}	string	"application/x-tar"
//	@Failure		400
//	@Failure		404
//	@Router			/api/admin/artifacts [get]
func getAdminArtifactsPlaceholder() {}

// GetArtifacts
//
//	@Summary		Get a job's kept artifacts
//	@Description	Returns the tar of files a testcase declared as artifacts,
//	@Description	for one of the caller's own submissions.
//	@Param			subId		query	number	true	"Submission id"
//	@Param			stage		query	string	true	"Stage name"
//	@Param			testcase	query	string	true	"Testcase name"
//	@Success		200			{file}	string	"application/x-tar"
//	@Failure		400
//	@Failure		404
//	@Router			/api/artifacts [get]
func (self *Handlers) GetArtifacts(
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

		span := trace.SpanFromContext(ctx)
		span.SetAttributes(
			attribute.Int64("submission.id", subId),
			attribute.String("job.stage", stageName),
			attribute.String("job.test", testName))

		submission, err := self.submissions.GetSubmission(ctx, subId)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusNotFound,
				Message:     "submission id not found",
				OriginError: err,
			}
		}

		if !slices.Contains(
			middlewares.NamespaceIdsFromContext(ctx), submission.NamespaceId) {
			return middlewares.HTTPError{
				StatusCode: http.StatusNotFound,
				Message:    "submission not in namespace",
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

		job := models.Job{SubId: subId, StageName: stageName, TestName: testName}
		fileName := job.ArtifactFilename()

		artifact, err := self.s3.GetFile(ctx, fileName)
		if err != nil {
			if errors.Is(err, types.FileNotExistError) {
				return middlewares.HTTPError{
					StatusCode:  http.StatusNotFound,
					Message:     "no artifacts kept for this testcase",
					OriginError: err,
				}
			}
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "failed to get artifacts",
				OriginError: err,
			}
		}

		w.Header().Set("Content-Type", "application/x-tar")
		w.Header().Set("Content-Disposition",
			fmt.Sprintf(`attachment; filename="%s"`, fileName))
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(artifact)
		return err
	}
}
