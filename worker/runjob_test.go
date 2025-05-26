package worker

import (
	"fmt"
	"testing"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

func TestRunJob(t *testing.T) {
	t.Run("Run user script should get output", func(t *testing.T) {
		ctx := t.Context()
		job := models.Job{
			TestContent: "echo hello, world\n",
			SoftTL:      30,
		}

		retch := make(chan int)
		jobin := make(chan models.Job, 1)
		jobout := make(chan models.Job, 1)
		jobin <- job
		go func() { fmt.Printf("RunJob\n"); RunJob(ctx, retch, jobin, jobout) }()
		if ret := <-retch; ret != STATUS_OK {
			t.Fatalf("expected no error, got %v, %+v", ret, job)
		}

		job = <-jobout
		expectedOutput := "hello, world\n"
		if expectedOutput != job.Stdout {
			t.Fatalf("expected %s, got %s", expectedOutput, job.Stdout)
		}
	})
}
