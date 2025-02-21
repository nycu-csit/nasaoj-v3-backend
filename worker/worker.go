package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	ctx := context.Background()
	THREADS := os.Getenv("THREADS")
	threads, err := strconv.Atoi(THREADS)

	if err != nil {
		fmt.Fprintf(os.Stderr, "threads strconv failed\n")
	}

	SCHEDULER_URL := os.Getenv("SCHEDULER_URL")
	PRESHARED_KEY := os.Getenv("PRESHARED_KEY")

	fmt.Printf("threads: %d\n", threads)

	c := make(chan int)
	for i := 0; i < threads; i++ {
		go RunThread(ctx, c, SCHEDULER_URL, PRESHARED_KEY)
	}
	for {
		ret := <-c
		go func() {
			if ret != 0 {
				time.Sleep(5 * time.Second)
			}
			go RunThread(ctx, c, SCHEDULER_URL, PRESHARED_KEY)
		}()
	}
}
