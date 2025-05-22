package main

import (
	"context"
	"fmt"
	"os"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/worker/env"
)

const (
	STATUS_OK                      = 0
	STATUS_HTTP_NEW_REQUEST_FAILED = -1
	STATUS_HTTP_GET_FAILED         = -2
	STATUS_HTTP_PUT_FAILED         = -3
	STATUS_JOB_QUEUE_EMPTY         = -5
	STATUS_BODY_READ_FAILED        = -6
	STATUS_EXEC_FAILED             = -7
	STATUS_EXEC_PREP_FAILED        = -8
	STATUS_CMD_OUTPUT_FAILED       = -9
	STATUS_JSON_MARSHAL_FAILED     = -10
	STATUS_TAR_PARSE_FAILED        = -11
	STATUS_DIR_CREATE_FAILED       = -12
	STATUS_FILE_WRITE_FAILED       = -13
)

func main() {
	ctx := context.Background()
	env.InitEnv()

	fmt.Printf("threads: %d\n", env.THREADS)

	jobin := make(chan models.JobWithFile, env.BUFFER_SIZE)
	jobout := make(chan models.JobWithFile, env.BUFFER_SIZE)
	go func() { // FetchJob threads
		retch := make(chan int)
		for {
			go FetchJob(ctx, retch, jobin)
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
			go PutJob(ctx, retch, jobout)
			ret := <-retch
			fmt.Fprintf(os.Stderr, "put thread end with status %v\n", ret)
		}
	}()

	retch := make(chan int)
	for i := int64(1); i < env.THREADS; i++ {
		go RunJob(ctx, retch, jobin, jobout)
	}
	for {
		go RunJob(ctx, retch, jobin, jobout)
		ret := <-retch
		fmt.Fprintf(os.Stderr, "run thread end with status %v\n", ret)
	}
}
