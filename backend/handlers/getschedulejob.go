package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

func GetScheduleJob(
	ctx context.Context,
	rdb *redis.Client,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		var err error
		fmt.Fprintf(os.Stderr, "GET /schedule/job\n")

		// Get Job from redis queue
		var jobkey string
		jobkey, err = rdb.RPop(ctx, "queue").Result()
		if err != nil {
			fmt.Fprintf(os.Stderr, "redis RPop failed\n")
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, "No job left in queue\n")
			return nil
		}

		var res string
		res, err = rdb.Get(ctx, jobkey).Result()
		if err != nil {
			fmt.Fprintf(os.Stderr, "redis Get failed\n")
			return nil
		}
		var jobs []models.Job
		err = json.Unmarshal([]byte(res), &jobs)
		if err != nil || len(jobs) < 1 {
			fmt.Fprintf(os.Stderr, "redis json unmarshal failed\n")
			return nil
		}
		job := jobs[0]

		// Reply
		payload, err := json.Marshal(&job)
		if err != nil {
			fmt.Printf("json parsing failed\n")
			return nil
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%s", string(payload[:]))

		return nil
	}
}
