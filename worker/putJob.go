package worker

import (
	"context"
	"fmt"
	"net/url"
	"os"

	"github.com/go-resty/resty/v2"
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

	client := resty.New()
	client.SetHeader("Authorization",
		"Bearer "+viper.GetString("access-token"))

	resultURL, err := url.JoinPath(
		viper.GetString("scheduler-url"), "/schedule/result")
	if err != nil {
		retCh <- fmt.Errorf("url construction failed, %w", err)
		return
	}

	job := <-jobch

	for i := 0; i < 3; i++ {
		resp, err := client.R().
			SetContext(ctx).
			SetBody(job.Job).
			Put(resultURL)

		if err == nil && resp.IsSuccess() {
			retCh <- nil
			return
		}
	}

	fmt.Fprintf(os.Stderr, "http put failed\n")
	retCh <- err
}
