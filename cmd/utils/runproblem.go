package utils

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/utils"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/worker"
	"gopkg.in/yaml.v2"
)

type jobInType struct {
	StageName string
	TestName  string
}

type jobOutType struct {
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
	color.Magenta("Run Stage %s\n", stageName)

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
		color.Yellow("job scheduled, job: %+v\n", jobInType{
			StageName: job.StageName,
			TestName:  job.TestName,
		})
		jobIn <- models.JobWithFile{
			Job:    job,
			Tarbuf: tarbuf.Bytes(),
		}
	}
	return nil
}

// Run problem locally
// Note that timeout is not implemented
var runProblemCmd = &cobra.Command{
	Use:   "run-problem",
	Short: "Run a single problem locally",
	Long:  `Run problem locally, the status and result will not `,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		execProblemId := viper.GetInt64("problem-id")
		mockArgs := make(map[string]string)
		for _, arg := range mockArgsRaw {
			strs := strings.Split(arg, "=")
			if len(strs) != 2 {
				return fmt.Errorf("argument format incorrect, %s\n", strs)
			}
			mockArgs[strs[0]] = strs[1]
		}

		color.Magenta("Problem Dir: %s\n", viper.GetString("problems-dir"))
		problems, err := utils.LoadProblems(viper.GetString("problems-dir"))
		if err != nil {
			return fmt.Errorf("load problems failed, %w\n", err)
		}
		color.Magenta("Testcases Dir: %s\n", viper.GetString("testcases-dir"))
		testcases, err := utils.LoadTestcases(viper.GetString("testcases-dir"))
		if err != nil {
			return fmt.Errorf("load testcases failed, %w\n", err)
		}

		var wg sync.WaitGroup

		jobIn := make(chan models.JobWithFile, viper.GetInt64("buffer-size"))
		jobOut := make(chan models.JobWithFile, viper.GetInt64("buffer-size"))
		retCh := make(chan error)
		for i := int64(0); i < viper.GetInt64("threads"); i++ {
			go worker.RunJob(ctx, retCh, jobIn, jobOut)
		}
		go func() {
			color.Magenta("Worker threads running\n")
			for {
				ret := <-retCh
				if ret != nil {
					color.Red("worker died with error, %w\n", ret)
				}
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

		pendingStages := make(chan string, viper.GetInt64("buffer-size"))
		for name, _ := range problem.Stages {
			if _, exist := stageDepCnt[name]; !exist {
				pendingStages <- name
			}
		}

		b, err := yaml.Marshal(problem)
		if err != nil {
			return fmt.Errorf("problem yaml marshal error, %w\n", err)
		}
		color.Cyan("Problem: %s\n", string(b[:]))

		go func() {
			color.Magenta("Output thread running\n")
			for {
				job := <-jobOut
				color.Yellow("job: %+v\n", jobOutType{
					StageName: job.StageName,
					TestName:  job.TestName,
					RetCode:   job.RetCode,
					ExecAt:    job.ExecAt,
				})

				color.Cyan("===== Script =====\n")
				color.White("%s\n", job.Script)
				color.Cyan("===== Stdout =====\n")
				color.White("%s\n", job.Stdout)
				color.Cyan("===== Stderr =====\n")
				color.White("%s\n", job.Stderr)
				color.Cyan("==================\n")

				if job.RetCode != 0 {
					color.Red("Job Failed\n")
					os.Exit(1)
				}

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
		color.Green("Job Succeed\n")
		return nil
	},
}

func init() {
	runProblemCmd.Flags().
		Int64("problem-id", 1, "Problem Id to test")
	runProblemCmd.MarkFlagRequired("problem-id")

	runProblemCmd.PersistentFlags().
		String("problems-dir", "./problems", "Directory to read problems")
	runProblemCmd.PersistentFlags().
		String("testcases-dir", "./testcases", "Directory to read testcases")
	runProblemCmd.PersistentFlags().
		String("running-dir", "/var/run/nasaoj-worker", "Worker running directory")
	runProblemCmd.PersistentFlags().
		Int64("buffer-size", 100, "Channel buffer size")
	runProblemCmd.PersistentFlags().
		Int64("threads", 1, "Thread count")
	runProblemCmd.Flags().
		StringSliceVarP(&mockArgsRaw,
			"arg", "a", []string{}, "Mock submission args")
}
