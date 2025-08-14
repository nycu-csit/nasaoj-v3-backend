package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

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

	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/schedule/job?tags=%s", viper.GetString("scheduler-url"),
			viper.GetString("tags")), nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "http new request failed\n")
		retCh <- err
		return
	}
	req.Header.Add("Authorization", "Bearer "+viper.GetString("access-token"))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "http GET request failed\n")
		retCh <- err
		return
	}
	if res.StatusCode == http.StatusServiceUnavailable {
		retCh <- fmt.Errorf("no job left in queue\n")
		return
	}

	b, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "body read failed\n")
		retCh <- err
		return
	}

	var job models.Job
	err = json.Unmarshal(b, &job)
	if err != nil {
		fmt.Fprintf(os.Stderr, "json unmarshal failed\n")
		retCh <- err
		return
	}

	req, err = http.NewRequestWithContext(ctx, "GET", job.FileURL, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "http new request failed\n")
		retCh <- err
		return
	}
	res, err = http.DefaultClient.Do(req)
	if err != nil || res.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "http GET request failed\n")
		retCh <- err
		return
	}

	b, err = io.ReadAll(res.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "body read failed\n")
		retCh <- err
		return
	}

	jobIn <- models.JobWithFile{
		Job:    job,
		Tarbuf: b,
	}
	retCh <- nil
}
