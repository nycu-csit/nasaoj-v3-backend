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

func RunThread(ctx context.Context, c chan int, scheduler string, psk string) {
	// Get job from scheduler
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/job", scheduler), nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "http new request failed\n")
		c <- -8787
		return
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", psk))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "http get failed\n")
		c <- -1
		return
	}
	if res.StatusCode == http.StatusServiceUnavailable {
		fmt.Fprintf(os.Stderr, "Job queue empty\n")
		c <- -503
		return
	}

	b, err := io.ReadAll(res.Body)

	if err != nil {
		fmt.Fprintf(os.Stderr, "body read failed\n")
		c <- -2
		return
	}

	var job models.Job
	err = json.Unmarshal(b, &job)
	fmt.Fprintf(os.Stderr, "job: %+v\n", job)

	// Execute TestContent
	cmd := exec.CommandContext(ctx, "bash", "-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "exec failed\n")
		c <- -3
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
		c <- -4
		return
	}

	job.Output = string(out[:])

	// Put results to scheduler
	b, err = json.Marshal(job)
	if err != nil {
		fmt.Fprintf(os.Stderr, "json marshal failed\n")
		c <- -5
		return
	}

	req, err = http.NewRequestWithContext(ctx, "PUT", fmt.Sprintf("%s/api/result", scheduler), bytes.NewReader(b))
	if err != nil {
		fmt.Fprintf(os.Stderr, "http new request failed, %v\n", err)
		c <- -8787
		return
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", psk))
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "http put failed\n")
		c <- -6
		return
	}

	c <- 0
}
