package workflows

import (
	"bytes"
	"context"
	"net/url"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.uber.org/zap"
)

// jobBasePath is a job's own working directory: ref/job tars are extracted
// here and the script runs with this as its cwd. Jobs run concurrently and
// each gets its own directory directly under running-dir - never a
// directory shared with any other job - and it is removed as soon as that
// one job finishes (see collectAndCleanup), independent of every other job
// in the submission.
func jobBasePath(job models.Job) string {
	return path.Join(viper.GetString("running-dir"), job.Id())
}

// submissionDir holds the one thing that *is* intentionally shared across a
// submission's jobs: the pool of artifacts already collected from completed
// stages (see submissionArtifactsDir), namespaced by submission so
// concurrent submissions of the same problem never collide. Removed
// wholesale once when the whole submission finishes (see RunSubmission),
// since a downstream stage may need to read an ancestor's artifacts at any
// point during the run.
func submissionDir(subId int64) string {
	return path.Join(viper.GetString("running-dir"), strconv.FormatInt(subId, 10))
}

// submissionArtifactsDir holds only the (typically small) files a testcase
// explicitly declared as artifacts - never a job's full working directory.
func submissionArtifactsDir(subId int64) string {
	return path.Join(submissionDir(subId), "artifacts")
}

// stageArtifactsDir is where a stage's collected artifacts are published so
// testcases in dependent stages can read them for the rest of the run.
func stageArtifactsDir(subId int64, stageName string) string {
	return path.Join(submissionArtifactsDir(subId), stageName)
}

// computeAncestors returns, for every stage, the set of stage names that
// must (transitively) complete before it can run, per Stage.Before edges.
// A downstream testcase's artifact visibility is scoped to exactly this set:
// concurrent, unrelated branches of the DAG never see each other's files.
func computeAncestors(stages map[string]models.Stage) map[string][]string {
	// parents[b] lists every stage that names b in its own Before, i.e. b's
	// direct dependencies.
	parents := make(map[string][]string, len(stages))
	for name, stage := range stages {
		for _, next := range stage.Before {
			parents[next] = append(parents[next], name)
		}
	}

	ancestors := make(map[string][]string, len(stages))
	for name := range stages {
		seen := make(map[string]bool)
		stack := append([]string{}, parents[name]...)
		for len(stack) > 0 {
			cur := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if seen[cur] {
				continue
			}
			seen[cur] = true
			stack = append(stack, parents[cur]...)
		}

		result := make([]string, 0, len(seen))
		for a := range seen {
			result = append(result, a)
		}
		ancestors[name] = result
	}
	return ancestors
}

// seedArtifacts copies every ancestor stage's already-collected artifacts
// into a job's working directory before it runs, so the job's script can
// read them exactly like any other input file - it doesn't need to know
// they came from a different testcase.
func seedArtifacts(
	ctx context.Context,
	job *models.JobWithFile,
	basepath string,
) error {
	for _, dir := range job.SeedDirs {
		if err := utils.CopyDir(ctx, basepath, dir); err != nil {
			return err
		}
	}
	return nil
}

// collectArtifacts archives the job's declared Artifacts out of its working
// directory (which is about to be deleted), uploads the archive straight to
// object storage via the presigned URL the backend issued at marshal time,
// and publishes the matched files into this stage's shared artifacts
// directory so dependent stages can read them for the rest of the run.
func collectArtifacts(
	ctx context.Context,
	job *models.JobWithFile,
	basepath string,
) {
	if len(job.Artifacts) == 0 {
		return
	}

	var tarbuf bytes.Buffer
	matched, err := utils.WriteDirTar(ctx, &tarbuf, basepath, job.Artifacts)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to archive artifacts",
				zap.String("job.id", job.Id()),
				zap.Error(err))
		return
	}
	if len(matched) == 0 {
		return
	}
	job.HasArtifacts = true

	if job.ArtifactUploadURL != "" {
		if err := uploadURL(ctx, job.ArtifactUploadURL, tarbuf.Bytes()); err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to upload artifacts",
					zap.String("job.id", job.Id()),
					zap.Error(err))
		}
	}

	dst := stageArtifactsDir(job.SubId, job.StageName)
	for _, rel := range matched {
		if err := utils.CopyFile(
			filepath.Join(dst, rel), filepath.Join(basepath, rel)); err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to publish artifact to shared pool",
					zap.String("job.id", job.Id()),
					zap.String("file", rel),
					zap.Error(err))
		}
	}
}

// uploadOutput uploads stdout/stderr directly to object storage via the
// presigned URLs the backend issued at marshal time - the backend never
// sees these bytes.
func uploadOutput(
	ctx context.Context,
	job *models.JobWithFile,
	stdout []byte,
	stderr []byte,
) {
	if job.StdoutUploadURL != "" {
		if err := uploadURL(ctx, job.StdoutUploadURL, stdout); err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to upload stdout",
					zap.String("job.id", job.Id()),
					zap.Error(err))
		}
	}
	if job.StderrUploadURL != "" {
		if err := uploadURL(ctx, job.StderrUploadURL, stderr); err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to upload stderr",
					zap.String("job.id", job.Id()),
					zap.Error(err))
		}
	}
}

// cleanupJobDir removes a job's working directory. Called once that job is
// fully done - its declared artifacts (if any) already collected - so
// nothing but the deliberately-kept files outlives the job itself.
func cleanupJobDir(
	ctx context.Context,
	job *models.JobWithFile,
	basepath string,
) {
	if err := utils.Rmrf(ctx, basepath); err != nil {
		tracing.Logger.Ctx(ctx).
			Warn("failed to clean up job working directory",
				zap.String("job.id", job.Id()),
				zap.String("dir", basepath),
				zap.Error(err))
	}
}

// uploadURL PUTs b to a presigned URL - the upload counterpart of
// downloadURL in fetchFiles.go.
func uploadURL(
	ctx context.Context,
	rawURL string,
	b []byte,
) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to parse upload url",
				zap.String("url", rawURL),
				zap.Error(err))
		return types.WrongFormatError
	}

	resp, err := resty.New().R().
		SetContext(ctx).
		SetQueryString(parsed.RawQuery).
		SetBody(b).
		Put(strings.Split(parsed.String(), "?")[0])
	if err != nil {
		return err
	}
	if !resp.IsSuccess() {
		tracing.Logger.Ctx(ctx).
			Error("failed to upload file",
				zap.String("url", parsed.String()),
				zap.Int("status", resp.StatusCode()))
		return types.HTTPRequestFailedError
	}
	return nil
}
