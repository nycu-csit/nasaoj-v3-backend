package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/worker/env"
)

func PutJob(
	pctx context.Context,
	ret chan int,
	jobch chan models.JobWithFile,
) {
	ctx, cancel := context.WithCancel(pctx)
	defer cancel()

	job := <-jobch
	b, err := json.Marshal(models.Job{
		SubId:     job.SubId,
		StageName: job.StageName,
		TestName:  job.TestName,

		Script: job.Script,

		FileURL: job.FileURL,
		HardTL:  job.HardTL,
		SoftTL:  job.SoftTL,

		Stdout:  job.Stdout,
		Stderr:  job.Stderr,
		RetCode: job.RetCode,
		ExecAt:  job.ExecAt,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "json marshal failed, %s\n", err)
		ret <- STATUS_JSON_MARSHAL_FAILED
		return
	}

	req, err := http.NewRequestWithContext(ctx, "PUT",
		fmt.Sprintf("%s/schedule/result", env.SCHEDULER_URL), bytes.NewReader(b))
	if err != nil {
		fmt.Fprintf(os.Stderr, "http new request failed, %s\n", err)
		ret <- STATUS_HTTP_NEW_REQUEST_FAILED
		return
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", env.PRESHARED_KEY))
	res, err := http.DefaultClient.Do(req)
	if err != nil || res.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "http put failed\n")
		ret <- STATUS_HTTP_PUT_FAILED
		return
	}

	ret <- STATUS_OK
}
