package api

import (
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const artifactsURLTimeout = 15 * time.Minute

type GetArtifactsURLOutput struct {
	URL string `json:"url"`
}

// GetArtifactsURL
//
//	@Summary		Get a presigned download URL for a job's kept artifacts (admin)
//	@Description	Presigns a short-lived MinIO URL so a caller (e.g. the admin
//	@Description	CLI) can download the artifact tar directly from object
//	@Description	storage instead of proxying the bytes through the backend.
//	@Description	Admin only - same access level as stderr.
//	@Param			subId		query		number	true	"Submission id"
//	@Param			stage		query		string	true	"Stage name"
//	@Param			testcase	query		string	true	"Testcase name"
//	@Success		200			{object}	GetArtifactsURLOutput
//	@Failure		400
//	@Failure		404
//	@Router			/api/admin/artifacts/url [get]
func (self *Handlers) GetArtifactsURL(
	w http.ResponseWriter, req bunrouter.Request,
) error {
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

	job := models.Job{SubId: subId, StageName: stageName, TestName: testName}
	fileName := job.ArtifactFilename()

	exist, err := self.s3.ExistFile(ctx, fileName)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "failed to check artifact file",
			OriginError: err,
		}
	}
	if !exist {
		return middlewares.HTTPError{
			StatusCode:  http.StatusNotFound,
			Message:     "no artifacts kept for this testcase",
			OriginError: types.FileNotExistError,
		}
	}

	url, err := self.s3.GetFileURL(ctx, fileName, artifactsURLTimeout)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "failed to presign artifact url",
			OriginError: err,
		}
	}

	return bunrouter.JSON(w, GetArtifactsURLOutput{URL: url})
}
