package test

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/implements"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/workflows"
)

var problemCmd = &cobra.Command{
	Use:   "problem",
	Short: "Run a problem on local",
	RunE: func(cmd *cobra.Command, args []string) error {
		color.Blue("Start running problem '%d'\n", viper.GetInt64("problem-id"))

		ctx := context.Background()
		problems, err := utils.LoadProblems(viper.GetString("problems-dir"))
		if err != nil {
			return fmt.Errorf("load problems failed, %w", err)
		}
		testcases, err := utils.LoadTestcases(viper.GetString("testcases-dir"))
		if err != nil {
			return fmt.Errorf("load testcases failed, %w", err)
		}

		problem, ok := problems[viper.GetInt64("problem-id")]
		if !ok {
			return fmt.Errorf("no such problem '%d'", viper.GetInt64("problem-id"))
		}

		mockArgs := make(map[string]string)
		for _, arg := range viper.GetStringSlice("arg") {
			strs := strings.Split(arg, "=")
			if len(strs) != 2 {
				return fmt.Errorf("argument format incorrect, %s\n", strs)
			}
			mockArgs[strs[0]] = strs[1]
		}

		// Initialize SQLite
		sqldb, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
		if err != nil {
			return fmt.Errorf("sqlite init failed, %w", err)
		}
		bunDB := bun.NewDB(sqldb, sqlitedialect.New())
		utils.InitDB(ctx, bunDB)

		db := implements.NewBunDatabase(bunDB)
		jobQue := implements.NewChanJobQueue()
		s3 := implements.NewMapObjectStorage()
		scoreCache := implements.NewVoidScoreCache()

		jobIn := make(chan models.JobWithFile, viper.GetInt64("buffer-size"))
		jobOut := make(chan models.JobWithFile, viper.GetInt64("buffer-size"))
		jobDone := make(chan models.JobWithFile, viper.GetInt64("buffer-size"))

		go workflows.WorkerThread(ctx, jobIn, jobOut)
		go workflows.OutputThread(ctx, jobOut, jobDone)
		go func() { // Marshal jobs with files
			for {
				job, err := jobQue.GetJob(ctx, problem.Tag)
				if err != nil { // jobQue closed
					// color.Red("get job from job queue failed, %v\n", err)
					close(jobIn)
					return
				}
				tarbuf, err := s3.GetFileByURL(ctx, job.FileURL)
				if err != nil {
					color.Red("get file failed, %v\n", err)
				}
				jobIn <- models.JobWithFile{
					Job:    job,
					Tarbuf: tarbuf,
				}
			}
		}()

		submission := models.Submission{
			ProblemName: problem.Name,
			ProblemId:   problem.ProblemId,

			CsId:     "114514",
			SubmitAt: time.Now().Unix(),
			Args:     mockArgs,

			Score: models.Score{
				Score:  0,
				Stages: make(map[string]int64),
			},
		}
		err = db.InsertSubmissionAndJobs(ctx, &submission, problem)
		if err != nil {
			return fmt.Errorf("insert submission failed, %w", err)
		}

		err = workflows.RunProblem(ctx, jobQue, db, s3,
			problems, testcases, submission)
		if err != nil {
			return fmt.Errorf("run problem failed, %w", err)
		}

		color.Cyan("workflows run problem done\n")

		isJobInClosed := false
		for {
			jobWithFile, ok := <-jobDone
			if !ok {
				break // channel closed
			}
			color.Cyan("job resolving deps\n")
			job := jobWithFile.Job

			if err = db.UpdateJob(ctx, &job); err != nil {
				return fmt.Errorf("job update failed, %w", err)
			}

			if job.Status == models.StatusSuccess {
				err = workflows.HandleStageDone(ctx, db, jobQue, s3, scoreCache,
					problems, testcases, job)
				if err != nil {
					return fmt.Errorf("handle job done failed, %w", err)
				}
			} else {
				if err = db.SkipJobs(ctx, job.SubId); err != nil {
					return fmt.Errorf("job skip status update failed, %w", err)
				}
			}

			if !isJobInClosed && jobQue.Size(problem.Tag) == 0 {
				color.Cyan("jobQue closed\n")
				jobQue.Close(problem.Tag)
				isJobInClosed = true
			}
		}

		color.Magenta("Program exit\n")
		return nil
	},
}

func init() {
	problemCmd.Flags().
		String("running-dir", "/tmp/nasaoj-worker", "Directory to run worker")

	problemCmd.Flags().
		Int64("problem-id", 1, "Problem Id to test")
	problemCmd.MarkFlagRequired("problem-id")
	problemCmd.Flags().
		Int64("buffer-size", 100, "Channel buffer size")
	problemCmd.Flags().
		StringSliceP("arg", "a", []string{}, "Mock submission args (key=value)")
}
