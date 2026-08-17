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
)

// GetArtifacts
//
//	@Summary		Get a job's kept artifacts (admin)
//	@Description	Returns the tar of files a testcase declared as artifacts,
//	@Description	for any submission. Admin only - artifacts may contain
//	@Description	build output/binaries not meant for students, so this is
//	@Description	kept at the same access level as stderr.
//	@Param			subId		query	number	true	"Submission id"
//	@Param			stage		query	string	true	"Stage name"
//	@Param			testcase	query	string	true	"Testcase name"
//	@Success		200			{file}	string	"application/x-tar"
//	@Failure		400
//	@Failure		404
//	@Router			/api/admin/artifacts [get]
func (self *Handlers) GetArtifacts(
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
