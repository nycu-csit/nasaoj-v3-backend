package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

const (
	STATUS_OK                      = 0
	STATUS_HTTP_NEW_REQUEST_FAILED = -1
	STATUS_HTTP_GET_FAILED         = -2
	STATUS_HTTP_PUT_FAILED         = -3
	STATUS_HTTP_HEAD_FAILED        = -4
	STATUS_JOB_QUEUE_EMPTY         = -5
	STATUS_BODY_READ_FAILED        = -6
	STATUS_EXEC_FAILED             = -7
	STATUS_EXEC_PREP_FAILED        = -8
	STATUS_CMD_OUTPUT_FAILED       = -9
	STATUS_JSON_MARSHAL_FAILED     = -10
)

func main() {
	ctx := context.Background()
	THREADS := os.Getenv("THREADS")
	threads, err := strconv.ParseInt(THREADS, 10, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "threads strconv failed\n")
		return
	}

	BUFFER_SIZE := os.Getenv("BUFFER_SIZE")
	buffer_size, err := strconv.ParseInt(BUFFER_SIZE, 10, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "buffer size strconv failed\n")
		return
	}

	THREAD_BACKOFF := os.Getenv("THREAD_BACKOFF")
	thread_backoff, err := strconv.ParseInt(THREAD_BACKOFF, 10, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "thread backoff strconv failed\n")
		return
	}

	SCHEDULER_URL := os.Getenv("SCHEDULER_URL")
	PRESHARED_KEY := os.Getenv("PRESHARED_KEY")

	fmt.Printf("threads: %d\n", threads)

	jobin := make(chan models.Job, buffer_size)
	jobout := make(chan models.Job, buffer_size)
	go func() { // FetchJob threads
		retch := make(chan int)
		for {
			go FetchJob(ctx, retch, jobin, SCHEDULER_URL, PRESHARED_KEY)
			ret := <-retch
			fmt.Fprintf(os.Stderr, "fetch thread end with status %v\n", ret)
			if ret != 0 {
				time.Sleep(time.Duration(thread_backoff) * time.Second)
			}
		}
	}()

	go func() { // PutJob threads
		retch := make(chan int)
		for {
			go PutJob(ctx, retch, jobout, SCHEDULER_URL, PRESHARED_KEY)
			ret := <-retch
			fmt.Fprintf(os.Stderr, "put thread end with status %v\n", ret)
		}
	}()

	retch := make(chan int)
	for i := int64(0); i < threads; i++ {
		go RunJob(ctx, retch, jobin, jobout)
	}
	for {
		ret := <-retch
		fmt.Fprintf(os.Stderr, "run thread end with status %v\n", ret)
		go RunJob(ctx, retch, jobin, jobout)
	}
}
