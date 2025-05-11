package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"text/template"

	"github.com/redis/go-redis/v9"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

func RunStage(
	ctx context.Context,
	rdb *redis.Client,
	problems ProblemRepository,
	testcases map[string]*template.Template,
	submission models.Submission,
	stageName string,
) error {
	var err error

	fmt.Fprintf(os.Stderr, "RunStage %s-%s\n", submission.ProblemName, stageName)

	problem := problems[int(submission.ProblemId)]
	args := make(map[string]string)
	maps.Copy(args, submission.Args)
	maps.Copy(args, problem.Stages[stageName].Args)

	fmt.Fprintf(os.Stderr, "Stage: %+v\n", problem.Stages[stageName])

	for _, testcase := range problem.Stages[stageName].Testcases {
		fmt.Fprintf(os.Stderr, "testcase: %v\n", testcase)
		// Record the different pairs
		diff := make(map[string]string)
		for key, val := range testcase.Args {
			if ori, exist := args[key]; exist {
				diff[key] = ori
			}
			args[key] = val
		}
		b := bytes.NewBufferString("")
		err = testcases[testcase.File].Execute(b, submission.Args)
		if err != nil {
			return fmt.Errorf("template execute failed, %s\n", err)
		}
		// Restore the different pairs
		for key, ori := range diff {
			args[key] = ori
		}

		job := models.Job{
			SubId:       submission.Id,
			StageName:   stageName,
			TestName:    testcase.Name,
			TestContent: b.String(),
		}
		fmt.Fprintf(os.Stderr, "job: %+v\n", job)

		payload, err := json.Marshal(job)
		if err != nil {
			return fmt.Errorf("job JSON Marshal error, %w\n", err)
		}
		fmt.Fprintf(os.Stderr, "job payload: %s\n", string(payload[:]))
		_, err = rdb.Set(ctx, job.Id(), payload, 0).Result()
		if err != nil {
			return fmt.Errorf("redis Set failed, %s\n", err)
		}
		_, err = rdb.LPush(ctx, "queue", job.Id()).Result()
		if err != nil {
			return fmt.Errorf("redis LPush queue failed, %s\n", err)
		}
	}

	return nil
}
