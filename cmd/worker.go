package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/worker"
)

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "NASAOJ worker server",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

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

		retCh := make(chan error)
		for i := int64(0); i < viper.GetInt64("threads"); i++ {
			go worker.RunJob(ctx, retCh, jobIn, jobOut)
		}
		for {
			ret := <-retCh
			if ret != nil {
				fmt.Fprintf(os.Stderr, "run thread end with error, %s\n", ret)
			}
			go worker.RunJob(ctx, retCh, jobIn, jobOut)
		}
	},
}

func init() {
	workerCmd.PersistentFlags().
		Int64("threads", 20, "Threads count")
	viper.BindPFlag("threads",
		workerCmd.PersistentFlags().Lookup("threads"))
	workerCmd.PersistentFlags().
		Int64("buffer-size", 100, "Channel buffer size")
	viper.BindPFlag("buffer-size",
		workerCmd.PersistentFlags().Lookup("buffer-size"))
	workerCmd.PersistentFlags().
		Int64("thread-backoff", 100, "Thread backoff seconds")
	viper.BindPFlag("thread-backoff",
		workerCmd.PersistentFlags().Lookup("thread-backoff"))
	workerCmd.PersistentFlags().
		String("scheduler-url", "http://localhost:8081", "Scheduler url")
	viper.BindPFlag("scheduler-url",
		workerCmd.PersistentFlags().Lookup("scheduler-url"))
	workerCmd.PersistentFlags().
		String("preshared-key", "poop", "Scheduler pre-shared key")
	viper.BindPFlag("preshared-key",
		workerCmd.PersistentFlags().Lookup("preshared-key"))
	workerCmd.PersistentFlags().
		String("tags", "default", "Tags for worker to take jobs")
	viper.BindPFlag("tags",
		workerCmd.PersistentFlags().Lookup("tags"))
	workerCmd.PersistentFlags().
		String("running-dir", "/var/run/nasaoj-worker", "Worker running directory")
	viper.BindPFlag("running-dir",
		workerCmd.PersistentFlags().Lookup("running-dir"))
}
