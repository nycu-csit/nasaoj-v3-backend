package workflows

import (
	"context"
	"errors"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"go.uber.org/zap"
)

// FetchJobFiles ensures the job's ref tar is present on disk (downloading it
// once per ref sha) and returns the job tar bytes. Used after marshalling a job.
func FetchJobFiles(
	ctx context.Context,
	job *models.Job,
) ([]byte, error) {
	refFilePath := path.Join(viper.GetString("running-dir"), job.RefFilename())
	if _, err := os.Stat(refFilePath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			tracing.Logger.Ctx(ctx).
				Error("failed to stat ref file",
					zap.String("refFile.path", refFilePath),
					zap.Error(err))
			return nil, err
		}

		refBody, err := downloadURL(ctx, job.RefFileURL)
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(refFilePath, refBody, 0644); err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to write ref file",
					zap.Error(err))
			return nil, err
		}
	}

	return downloadURL(ctx, job.JobFileURL)
}

func downloadURL(
	ctx context.Context,
	rawURL string,
) ([]byte, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to parse file url",
				zap.String("url", rawURL),
				zap.Error(err))
		return nil, types.WrongFormatError
	}

	resp, err := resty.New().R().
		SetContext(ctx).
		SetQueryString(parsed.RawQuery).
		Get(strings.Split(parsed.String(), "?")[0])
	if err != nil {
		return nil, err
	}
	if !resp.IsSuccess() {
		tracing.Logger.Ctx(ctx).
			Error("failed to download file",
				zap.String("url", parsed.String()),
				zap.Int("status", resp.StatusCode()))
		return nil, types.HTTPRequestFailedError
	}
	return resp.Body(), nil
}
