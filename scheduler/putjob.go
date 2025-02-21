package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/ytshih/na-judge/models"
)

func PutJob(ctx context.Context, rdb *redis.Client) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		body, err := ioutil.ReadAll(req.Body)

		if err != nil {
			fmt.Printf("IO failed\n")
			return nil
		}

		fmt.Fprintf(os.Stderr, "Raw body: %s\n", string(body[:]))
		var job models.Job
		err = json.Unmarshal(body, &job)
		if err != nil {
			fmt.Fprintf(os.Stderr, "json unmarshal failed, %v\n", err)
			return nil
		}

		fmt.Printf("PUT /api/job\npayload: %+v\n", job)
		_, err = rdb.JSONSet(ctx, strconv.Itoa(job.Id), "$", job).Result()
		if err != nil {
			fmt.Fprintf(os.Stderr, "redis json set failed: %v\n", err)
			return nil
		}
		_, err = rdb.LPush(ctx, "queue", strconv.Itoa(job.Id)).Result()
		if err != nil {
			fmt.Fprintf(os.Stderr, "redis queue push failed\n")
			return nil
		}

		w.WriteHeader(http.StatusOK)
		return nil
	}
}
