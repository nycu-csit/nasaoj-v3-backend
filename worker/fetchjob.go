package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/worker/env"
)

func FetchJob(
	pctx context.Context,
	ret chan int,
	jobch chan models.JobWithFile,
) {
	ctx, cancel := context.WithCancel(pctx)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/schedule/job?tags=%s", env.SCHEDULER_URL, env.TAGS), nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "http new request failed\n")
		ret <- STATUS_HTTP_NEW_REQUEST_FAILED
		return
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", env.PRESHARED_KEY))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "http GET request failed\n")
		ret <- STATUS_HTTP_GET_FAILED
		return
	}
	if res.StatusCode == http.StatusServiceUnavailable {
		ret <- STATUS_JOB_QUEUE_EMPTY
		return
	}

	b, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "body read failed\n")
		ret <- STATUS_BODY_READ_FAILED
		return
	}

	var job models.Job
	err = json.Unmarshal(b, &job)
	if err != nil {
		fmt.Fprintf(os.Stderr, "json unmarshal failed\n")
		ret <- STATUS_JSON_MARSHAL_FAILED
		return
	}

	req, err = http.NewRequestWithContext(ctx, "GET", job.FileURL, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "http new request failed\n")
		ret <- STATUS_HTTP_NEW_REQUEST_FAILED
		return
	}
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "http GET request failed\n")
		ret <- STATUS_HTTP_GET_FAILED
		return
	}

	b, err = io.ReadAll(res.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "body read failed\n")
		ret <- STATUS_BODY_READ_FAILED
		return
	}

	jobch <- models.JobWithFile{
		Job:    job,
		Tarbuf: b,
	}
	ret <- STATUS_OK
}
