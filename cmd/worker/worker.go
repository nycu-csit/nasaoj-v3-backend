package worker

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/worker"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/worker/env"
)

var WorkerCmd = &cobra.Command{
	Use:   "worker",
	Short: "NASAOJ worker server",
	Run: func(cmd *cobra.Command, args []string) {
		env.InitEnv()

		ctx := context.Background()

		jobin := make(chan models.JobWithFile, env.BUFFER_SIZE)
		jobout := make(chan models.JobWithFile, env.BUFFER_SIZE)
		go func() { // FetchJob threads
			retch := make(chan int)
			for {
				go worker.FetchJob(ctx, retch, jobin)
				ret := <-retch
				fmt.Fprintf(os.Stderr, "fetch thread end with status %v\n", ret)
				if ret != 0 {
					time.Sleep(time.Duration(env.THREAD_BACKOFF) * time.Second)
				}
			}
		}()

		go func() { // PutJob threads
			retch := make(chan int)
			for {
				go worker.PutJob(ctx, retch, jobout)
				ret := <-retch
				fmt.Fprintf(os.Stderr, "put thread end with status %v\n", ret)
			}
		}()

		retch := make(chan int)
		for i := int64(1); i < env.THREADS; i++ {
			go worker.RunJob(ctx, retch, jobin, jobout)
		}
		for {
			go worker.RunJob(ctx, retch, jobin, jobout)
			ret := <-retch
			fmt.Fprintf(os.Stderr, "run thread end with status %v\n", ret)
		}
	},
}
