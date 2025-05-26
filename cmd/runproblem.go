package cmd

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	backendEnv "gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/env"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/utils"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/worker"
	workerEnv "gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/worker/env"
)

type jobOutput struct {
	StageName string
	TestName  string
	RetCode   int
	ExecAt    int64
}

var mockArgsRaw []string

func runStage(
	wg *sync.WaitGroup,
	jobIn chan models.JobWithFile,
	mockArgs map[string]string,
	problem models.Problem,
	stageName string,
	testcases utils.TestcaseRepository,
) error {
	stage := problem.Stages[stageName]
	for _, testcase := range stage.Testcases {
		contents := []models.Content{
			problem.PreScript,
			stage.PreScript,
			testcase.Content,
			stage.PostScript,
			problem.PostScript,
		}

		jobScript := ""
		for _, content := range contents {
			for _, script := range content.Scripts {
				jobScript += script + "\n"
			}
		}

		args := utils.ArgsOverlay(
			mockArgs,
			problem.Args,
			stage.Args,
			testcase.Args,
		)

		var tarbuf bytes.Buffer
		err := utils.WriteTar(&tarbuf, contents, testcases, args)
		if err != nil {
			return fmt.Errorf("write to tar failed, %w\n", err)
		}

		job := models.Job{
			StageName: stageName,
			TestName:  testcase.Name,

			Script: jobScript,

			HardTL: testcase.HardTL,
			SoftTL: testcase.SoftTL,
		}

		wg.Add(1)
		jobIn <- models.JobWithFile{
			Job:    job,
			Tarbuf: tarbuf.Bytes(),
		}
	}
	return nil
}

// Run problem locally
// Note that timeout is not implemented
var RunProblemCmd = &cobra.Command{
	Use:   "run-problem",
	Short: "Run a single problem for testing purpose",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		backendEnv.InitEnv()
		workerEnv.InitEnv()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		execProblemId, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("parse problem id failed, %w\n", err)
		}
		mockArgs := make(map[string]string)
		for _, arg := range mockArgsRaw {
			strs := strings.Split(arg, "=")
			if len(strs) != 2 {
				return fmt.Errorf("argument format incorrect, %s\n", strs)
			}
			mockArgs[strs[0]] = strs[1]
		}

		problems, err := utils.LoadProblems(backendEnv.PROBLEMS_DIR)
		if err != nil {
			return fmt.Errorf("load problems failed, %w\n", err)
		}
		testcases, err := utils.LoadTestcases(backendEnv.TESTCASES_DIR)
		if err != nil {
			return fmt.Errorf("load testcases failed, %w\n", err)
		}

		var wg sync.WaitGroup

		jobIn := make(chan models.JobWithFile, workerEnv.BUFFER_SIZE)
		jobOut := make(chan models.JobWithFile, workerEnv.BUFFER_SIZE)
		retCh := make(chan int)
		for i := int64(0); i < workerEnv.THREADS; i++ {
			go worker.RunJob(ctx, retCh, jobIn, jobOut)
		}
		go func() {
			fmt.Printf("Worker threads running\n")
			for {
				_ = <-retCh
				go worker.RunJob(ctx, retCh, jobIn, jobOut)
			}
		}()

		problem := problems[execProblemId]

		stageJobCnt := make(map[string]int64)
		stageDepCnt := make(map[string]int64)
		for name, stage := range problem.Stages {
			stageJobCnt[name] = int64(len(stage.Testcases))
			for _, next := range stage.Before {
				stageDepCnt[next] += 1
			}
		}

		pendingStages := make(chan string, workerEnv.BUFFER_SIZE)
		for name, _ := range problem.Stages {
			if _, exist := stageDepCnt[name]; !exist {
				pendingStages <- name
			}
		}

		go func() {
			fmt.Printf("Output thread running\n")
			for {
				job := <-jobOut
				color.HiYellow("job: %+v\n", jobOutput{
					StageName: job.StageName,
					TestName:  job.TestName,
					RetCode:   job.RetCode,
					ExecAt:    job.ExecAt,
				})

				color.Cyan("===== Stdout =====\n")
				fmt.Printf("%s\n", job.Stdout)
				color.Cyan("===== Stderr =====\n")
				fmt.Printf("%s\n", job.Stderr)
				color.Cyan("==================\n")

				stageJobCnt[job.StageName] -= 1
				if stageJobCnt[job.StageName] == 0 {
					for _, next := range problem.Stages[job.StageName].Before {
						stageDepCnt[next] -= 1
						if stageDepCnt[next] == 0 {
							pendingStages <- next
						}
					}
					delete(stageJobCnt, job.StageName)
				}
				wg.Done()
			}
		}()

		stageRunCnt := 0
		for stageRunCnt < len(problem.Stages) {
			stage := <-pendingStages
			err = runStage(&wg, jobIn, mockArgs, problem, stage, testcases)
			if err != nil {
				return fmt.Errorf("runStage failed, %w\n", err)
			}
			stageRunCnt += 1
		}

		wg.Wait()
		return nil
	},
}

func init() {
	RunProblemCmd.Flags().
		StringSliceVarP(&mockArgsRaw,
			"arg", "a", []string{""}, "Mock submission args")
}
