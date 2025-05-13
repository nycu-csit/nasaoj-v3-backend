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
			w.WriteHeader(http.StatusNoContent)
			fmt.Fprintf(w, "no content\n")
			return nil
		}
		var job models.Job
		err = json.Unmarshal([]byte(res), &job)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "redis json unmarshal failed\n")
			return fmt.Errorf("redis json unmarshal failed, %w\n", err)
		}

		// Reply
		payload, err := json.Marshal(&job)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "json parsing failed\n")
			return fmt.Errorf("json parsing failed\n")
		}

		_, err = rdb.SAdd(ctx, "jobset", job.Id()).Result()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "redis ZADD failed\n")
			return fmt.Errorf("redis ZADD failed, %w\n", err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%s", string(payload[:]))
		return nil
	}
}
