package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

func PutJob(
	pctx context.Context,
	retCh chan error,
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
		retCh <- err
		return
	}

	req, err := http.NewRequestWithContext(ctx, "PUT",
		fmt.Sprintf("%s/schedule/result", viper.GetString("scheduler-url")),
		bytes.NewReader(b))
	if err != nil {
		fmt.Fprintf(os.Stderr, "http new request failed, %s\n", err)
		retCh <- err
		return
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization",
		fmt.Sprintf("Bearer %s", viper.GetString("preshared-key")))
	res, err := http.DefaultClient.Do(req)
	if err != nil || res.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "http put failed\n")
		retCh <- err
		return
	}

	retCh <- nil
}
