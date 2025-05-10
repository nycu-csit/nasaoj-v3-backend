package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

const (
	STATUS_OK                      = 0
	STATUS_HTTP_NEW_REQUEST_FAILED = -8787
	STATUS_HTTP_GET_FAILED         = -1
	STATUS_JOB_QUEUE_EMPTY         = -503
	STATUS_BODY_READ_FAILED        = -2
	STATUS_EXEC_FAILED             = -3
	STATUS_CMD_OUTPUT_FAILED       = -4
	STATUS_JSON_MARSHAL_FAILED     = -5
	STATUS_HTTP_PUT_FAILED         = -6
)

func RunThread(
	ctx context.Context,
	c chan int,
	scheduler string,
	psk string,
) {
	// Get job from scheduler
	req, err := http.NewRequestWithContext(
		ctx, "GET", fmt.Sprintf("%s/schedule/job", scheduler), nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "http new request failed\n")
		c <- STATUS_HTTP_NEW_REQUEST_FAILED
		return
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", psk))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "http get failed\n")
		c <- STATUS_HTTP_GET_FAILED
		return
	}
	if res.StatusCode == http.StatusServiceUnavailable {
		fmt.Fprintf(os.Stderr, "Job queue empty\n")
		c <- STATUS_JOB_QUEUE_EMPTY
		return
	}

	b, err := io.ReadAll(res.Body)

	if err != nil {
		fmt.Fprintf(os.Stderr, "body read failed\n")
		c <- STATUS_BODY_READ_FAILED
		return
	}

	var job models.Job
	err = json.Unmarshal(b, &job)
	fmt.Fprintf(os.Stderr, "job: %+v\n", job)

	// Execute TestContent
	cmd := exec.CommandContext(ctx, "bash", "-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "exec failed\n")
		c <- STATUS_EXEC_FAILED
		return
	}

	stdin, err := cmd.StdinPipe()

	go func() {
		defer stdin.Close()

		io.WriteString(stdin, job.TestContent)
	}()

	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cmd output failed, %v\n", err)
		c <- STATUS_CMD_OUTPUT_FAILED
		return
	}

	job.Output = string(out[:])

	// Put results to scheduler
	b, err = json.Marshal(job)
	if err != nil {
		fmt.Fprintf(os.Stderr, "json marshal failed\n")
		c <- STATUS_JSON_MARSHAL_FAILED
		return
	}

	req, err = http.NewRequestWithContext(
		ctx, "PUT", fmt.Sprintf("%s/schedule/result", scheduler), bytes.NewReader(b))
	if err != nil {
		fmt.Fprintf(os.Stderr, "http new request failed, %v\n", err)
		c <- STATUS_HTTP_NEW_REQUEST_FAILED
		return
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", psk))
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "http put failed\n")
		c <- STATUS_HTTP_PUT_FAILED
		return
	}

	c <- STATUS_OK
}
