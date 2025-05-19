package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

func FetchJob(
	pctx context.Context,
	ret chan int,
	jobch chan models.Job,
	scheduler string,
	psk string,
) {
	ctx, cancel := context.WithCancel(pctx)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "HEAD",
		fmt.Sprintf("%s/schedule/job", scheduler), nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "http new request failed\n")
		ret <- STATUS_HTTP_NEW_REQUEST_FAILED
		return
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", psk))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "http HEAD request failed\n")
		ret <- STATUS_HTTP_HEAD_FAILED
		return
	}
	if res.StatusCode == http.StatusServiceUnavailable {
		ret <- STATUS_JOB_QUEUE_EMPTY
		return
	}

	req, err = http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/schedule/job", scheduler), nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "http new request failed\n")
		ret <- STATUS_HTTP_NEW_REQUEST_FAILED
		return
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", psk))
	res, err = http.DefaultClient.Do(req)
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

	jobch <- job
	ret <- STATUS_OK
}
