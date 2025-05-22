package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

func GetScheduleJob(
	ctx context.Context,
	rdb *redis.Client,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		var err error
		tags := req.URL.Query().Get("tags")
		fmt.Fprintf(os.Stderr, "GET /schedule/job?%s\n", tags)

		// Get Job from redis queue
		var jobkey string
		for _, tag := range strings.Split(tags, ",") {
			jobkey, err = rdb.RPop(ctx, fmt.Sprintf("queue-%s", tag)).Result()
			if err == nil {
				break
			}
		}
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusServiceUnavailable,
				Message:     "no job left in queue",
				OriginError: err,
			}
		}

		var res string
		res, err = rdb.Get(ctx, jobkey).Result()
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusNoContent,
				Message:     "no content",
				OriginError: err,
			}
		}
		var job models.Job
		err = json.Unmarshal([]byte(res), &job)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "redis json unmarshal failed",
				OriginError: err,
			}
		}

		// Reply
		payload, err := json.Marshal(&job)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "json parsing failed",
				OriginError: err,
			}
		}

		_, err = rdb.SAdd(ctx, "jobset", job.Id()).Result()
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "redis ZADD failed",
				OriginError: err,
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%s", string(payload[:]))
		return nil
	}
}
