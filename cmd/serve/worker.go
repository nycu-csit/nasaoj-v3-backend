package serve

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/worker"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/workflows"
)

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "NASAOJ worker server",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		// Initialize Tracing
		appname := "nasaoj-worker"
		tracing.InitTracer(appname)
		tracing.InitUptrace(appname)
		defer tracing.DeferUptrace(ctx)

		jobIn := make(chan models.JobWithFile, viper.GetInt64("buffer-size"))
		jobOut := make(chan models.JobWithFile, viper.GetInt64("buffer-size"))
		go func() { // FetchJob threads
			retCh := make(chan error)
			for {
				go worker.FetchJob(ctx, retCh, jobIn)
				ret := <-retCh
				if ret != nil {
					fmt.Fprintf(os.Stderr, "fetch thread end with error, %s\n", ret)
					time.Sleep(time.Duration(
						viper.GetInt64("thread-backoff")) * time.Second)
				}
			}
		}()

		go func() { // PutJob threads
			retCh := make(chan error)
			for {
				go worker.PutJob(ctx, retCh, jobOut)
				ret := <-retCh
				if ret != nil {
					fmt.Fprintf(os.Stderr, "put thread end with error, %s\n", ret)
				}
			}
		}()

		for i := int64(0); i < viper.GetInt64("threads"); i++ {
			go workflows.WorkerThread(ctx, jobIn, jobOut)
		}

		select {
		case <-ctx.Done():
		}
	},
}

func init() {
	workerCmd.Flags().
		Int64("threads", 20, "Threads count")
	workerCmd.Flags().
		Int64("buffer-size", 100, "Channel buffer size")
	workerCmd.Flags().
		Int64("thread-backoff", 10, "Thread backoff seconds")
	workerCmd.Flags().
		Int64("putjob-retry", 3, "How many times to retry if put job failed")
	workerCmd.Flags().
		String("scheduler-url", "http://127.0.0.1:8080", "Scheduler url")
	workerCmd.Flags().
		String("access-token", "", "Scheduler access token")
	workerCmd.Flags().
		String("tags", "default", "Tags for worker to take jobs")
	workerCmd.Flags().
		String("running-dir", "/var/run/nasaoj-worker", "Worker running directory")
	workerCmd.Flags().
		String("uptrace-dsn", "", "Uptrace DSN")
}
