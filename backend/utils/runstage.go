package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/template"
	"time"

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

	problem := problems[submission.ProblemId]
	stage := problem.Stages[stageName]

	fmt.Fprintf(os.Stderr, "Stage: %+v\n", problem.Stages[stageName])

	for _, testcase := range stage.Testcases {
		fmt.Fprintf(os.Stderr, "testcase: %v\n", testcase)
		testContent := ""

		args := ArgsOverlay(
			submission.Args,
			problem.Args,
			stage.Args,
			testcase.Args,
		)

		// Pre-scripts
		for _, preScriptName := range problem.PreScripts {
			preScriptTmpl, ok := testcases[preScriptName]
			if !ok {
				return fmt.Errorf("pre-script %s not exist\n", preScriptName)
			}

			b := bytes.NewBufferString("")
			err = preScriptTmpl.Execute(b, args)
			if err != nil {
				return fmt.Errorf("template execute failed, %w\n", err)
			}

			testContent += b.String()
		}

		// Testcase
		testcaseTmpl, ok := testcases[testcase.File]
		if !ok {
			return fmt.Errorf("testcase %s not exist\n", testcase.File)
		}

		b := bytes.NewBufferString("")
		err = testcaseTmpl.Execute(b, args)
		if err != nil {
			return fmt.Errorf("template execute failed, %w\n", err)
		}

		testContent += b.String()

		// Post-scripts
		for _, postScriptName := range problem.PostScripts {
			postScriptTmpl, ok := testcases[postScriptName]
			if !ok {
				return fmt.Errorf("post-script %s not exist\n", postScriptName)
			}

			b := bytes.NewBufferString("")
			err = postScriptTmpl.Execute(b, args)
			if err != nil {
				return fmt.Errorf("template execute failed, %w\n", err)
			}

			testContent += b.String()
		}

		job := models.Job{
			SubId:       submission.Id,
			StageName:   stageName,
			TestName:    testcase.Name,
			TestContent: testContent,
			HardTL:      testcase.HardTL,
			SoftTL:      testcase.SoftTL,
		}
		fmt.Fprintf(os.Stderr, "job: %+v\n", job)

		payload, err := json.Marshal(job)
		if err != nil {
			return fmt.Errorf("job JSON Marshal error, %w\n", err)
		}
		fmt.Fprintf(os.Stderr, "job %s payload: %s\n", job.Id(), string(payload[:]))
		_, err = rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
			_ = pipe.Set(ctx, job.Id(), payload, 0)
			_ = pipe.LPush(ctx, "queue", job.Id())

			subJob := fmt.Sprintf("sub-job-%s", submission.Id)
			_ = pipe.HSet(ctx, subJob, job.Id(), "")
			duration := time.Duration(job.HardTL) * time.Second
			_ = pipe.HExpire(ctx, subJob, duration, job.Id())
			return nil
		})
		if err != nil {
			return fmt.Errorf("redis pipeline operation failed, %w\n", err)
		}
	}

	return nil
}
