package worker

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

func FetchJob(
	pctx context.Context,
	retCh chan error,
	jobIn chan models.JobWithFile,
) {
	ctx, cancel := context.WithCancel(pctx)
	defer cancel()

	client := resty.New()

	jobURL, err := url.JoinPath(
		viper.GetString("scheduler-url"), "/schedule/job")
	if err != nil {
		retCh <- fmt.Errorf("url construction failed, %w", err)
		return
	}

	var job models.Job
	resp, err := client.R().
		SetHeader("Authorization",
			"Bearer "+viper.GetString("access-token")).
		SetContext(ctx).
		SetResult(&job).
		ForceContentType("application/json").
		SetQueryParam("tags", viper.GetString("tags")).
		Get(jobURL)

	if resp.StatusCode() == http.StatusServiceUnavailable {
		retCh <- fmt.Errorf("no job left in queue")
		return
	}

	if !resp.IsSuccess() {
		retCh <- fmt.Errorf("GET %s failed, %v", jobURL, resp.Error())
		return
	}

	fileURL, err := url.Parse(job.FileURL)
	if err != nil {
		retCh <- fmt.Errorf("file url parse failed, %w", err)
		return
	}
	resp, err = client.R().
		SetContext(ctx).
		SetQueryString(fileURL.RawQuery).
		Get(strings.Split(fileURL.String(), "?")[0])

	if err != nil {
		retCh <- err
		return
	}

	if !resp.IsSuccess() {
		// fmt.Fprintf(os.Stderr, "%##+v\n", resp.Request)
		// fmt.Fprintf(os.Stderr, "%##+v\n", resp.RawResponse)
		retCh <- fmt.Errorf("GET %s failed, %v",
			fileURL.String(), resp.Error())
		return
	}

	jobIn <- models.JobWithFile{
		Job:    job,
		Tarbuf: resp.Body(),
	}
	retCh <- nil
}
