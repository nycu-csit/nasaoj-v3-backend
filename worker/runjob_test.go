package worker

import (
	"archive/tar"
	"bytes"
	"testing"

	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

func TestRunJob(t *testing.T) {
	t.Run("Run user script should get output", func(t *testing.T) {
		ctx := t.Context()
		viper.Set("running-dir", "/tmp/nasaoj-worker")

		var tarbuf bytes.Buffer
		tw := tar.NewWriter(&tarbuf)
		if err := tw.Close(); err != nil {
			t.Fatalf("tar writer close failed, %s\n", err)
		}

		job := models.JobWithFile{
			Job: models.Job{
				SubId:     114514,
				StageName: "stage-1919",
				TestName:  "test-810",

				Script: "echo hello, world\n",

				HardTL: 30,
				SoftTL: 25,
			},
			Tarbuf: tarbuf.Bytes(),
		}

		retCh := make(chan error, 1)
		jobIn := make(chan models.JobWithFile, 1)
		jobOut := make(chan models.JobWithFile, 1)

		jobIn <- job
		go func() {
			RunJob(ctx, retCh, jobIn, jobOut)
		}()
		if ret := <-retCh; ret != nil {
			t.Fatalf("expected no error, got %v, %+v", ret, job.Job)
		}

		job = <-jobOut
		expectedOutput := "hello, world\n"
		if expectedOutput != job.Stdout {
			t.Fatalf("expected %s, got %s", expectedOutput, job.Stdout)
		}
	})
}
