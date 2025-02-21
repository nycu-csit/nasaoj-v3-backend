package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"text/template"

	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

func GetJob(ctx context.Context, rdb *redis.Client) func(http.ResponseWriter, bunrouter.Request) error {
	TESTCASES_DIR := os.Getenv("TESTCASES_DIR")

	return func(w http.ResponseWriter, req bunrouter.Request) error {
		var err error
		fmt.Printf("GET /api/job\n")

		// Get Job from redis queue
		var jobId string
		jobId, err = rdb.RPop(ctx, "queue").Result()
		if err != nil {
			fmt.Fprintf(os.Stderr, "redis RPop failed\n")
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, "No job left in queue\n")
			return nil
		}

		var res string
		res, err = rdb.JSONGet(ctx, jobId, "$").Result()
		if err != nil {
			fmt.Fprintf(os.Stderr, "redis JSONGet failed\n")
			return nil
		}
		var jobs []models.Job
		err = json.Unmarshal([]byte(res), &jobs)
		if err != nil || len(jobs) < 1 {
			fmt.Fprintf(os.Stderr, "redis json unmarshal failed\n")
			return nil
		}
		job := jobs[0]

		// Fetch Content
		if job.TestContent == "" {
			file, err := os.ReadFile(path.Join(TESTCASES_DIR, job.TestName))

			if err != nil {
				fmt.Fprintf(os.Stderr, "file read failed\n")
				return nil
			}

			job.TestContent = string(file[:])
		}

		// Templating
		tmpl, err := template.New("content").Parse(job.TestContent)
		if err != nil {
			fmt.Fprintf(os.Stderr, "templating parse failed\n")
			return nil
		}
		buf := bytes.NewBufferString("")
		err = tmpl.Execute(buf, job.Args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "template execute failed\n")
			return nil
		}
		job.TestContent = buf.String()

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
