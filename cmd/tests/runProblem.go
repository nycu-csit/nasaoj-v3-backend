package tests

import (
	"database/sql"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
	"github.com/uptrace/bun/extra/bunotel"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/implements"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/workflows"
	"go.uber.org/zap"
)

var problemCmd = &cobra.Command{
	Use:   "problem",
	Short: "Run a problem on local environment",
	Run: func(cmd *cobra.Command, args []string) {
		color.Blue("Start running problem '%d'\n", viper.GetInt64("problem-id"))

		ctx := cmd.Context()
		if viper.GetString("uptrace-dsn") != "" {
			tracing.InitUptrace(viper.GetString("uptrace-dsn"))
			defer tracing.DeferUptrace(ctx)
		}

		mockArgs := make(map[string]string)
		for _, arg := range viper.GetStringSlice("arg") {
			strs := strings.Split(arg, "=")
			if len(strs) != 2 {
				tracing.Logger.Ctx(ctx).
					Panic("argument format incorrect")
				panic(types.WrongFormatError)
			}
			mockArgs[strs[0]] = strs[1]
		}

		// Initialize SQLite
		sqldb, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to init sqlite",
					zap.Error(err))
			panic(err)
		}
		bunDB := bun.NewDB(sqldb, sqlitedialect.New())
		bunDB.AddQueryHook(bunotel.NewQueryHook(bunotel.WithDBName("test")))
		if err := utils.InitDB(ctx, bunDB); err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to init db schema",
					zap.Error(err))
			panic(err)
		}

		db := implements.NewBunDatabase(bunDB)
		jobQue := implements.NewChanJobQueue()
		s3 := implements.NewMapObjectStorage()
		problems := implements.NewDBProblemStorage(db)
		tokens := implements.NewMapTokenStorage()

		err = utils.LoadProblems(ctx, problems,
			viper.GetString("problems-dir"),
			viper.GetString("testcases-dir"))
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to load problems",
					zap.Error(err))
			panic(err)
		}

		err = utils.LoadProblems(ctx, problems,
			viper.GetString("tools-dir"),
			viper.GetString("testcases-dir"))
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to load tools",
					zap.Error(err))
			panic(err)
		}

		if err := db.EnableAllProblem(ctx); err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to enable problems",
					zap.Error(err))
			panic(err)
		}

		problem, err := problems.GetProblemById(
			ctx, viper.GetInt64("problem-id"))
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("problem not exist",
					zap.Error(err))
			panic(err)
		}
		problemMarshaled, err := problem.Marshal()
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to marshal problem",
					zap.Error(err))
			panic(err)
		}

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

				timeout := time.Duration(job.HardTL) * time.Second
				job.FileURL, err = s3.GetFileURL(ctx, job.Filename(), timeout)
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
			RefId:   problem.ProblemId,
			RefSHA1: problemMarshaled.SHA1,

			CsId:      "114514",
			CreatedAt: time.Now().Unix(),
			Args:      mockArgs,

			Score: models.Score{
				Score:  0,
				Stages: make(map[string]int64),
			},
		}
		err = db.InsertProblemSubmission(ctx, &submission, problem)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to insert submission",
					zap.Error(err))
			panic(err)
		}

		err = workflows.RunProblem(
			ctx, jobQue, db, s3, problems, tokens, submission)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to run problem",
					zap.Error(err))
			panic(err)
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
				tracing.Logger.Ctx(ctx).
					Panic("failed to update job",
						zap.Error(err))
				panic(err)
			}

			if job.Status == models.StatusSuccess {
				err = workflows.HandleStageDone(
					ctx, db, jobQue, s3, problems, tokens, job)
				if err != nil {
					tracing.Logger.Ctx(ctx).
						Panic("failed to handle job done",
							zap.Error(err))
					panic(err)
				}
			} else {
				if err = db.SkipJobs(ctx, job.SubId); err != nil {
					tracing.Logger.Ctx(ctx).
						Panic("failed to update job skip status",
							zap.Error(err))
					panic(err)
				}
			}

			if !isJobInClosed && jobQue.Size(problem.Tag) == 0 {
				color.Cyan("jobQue closed\n")
				jobQue.Close(problem.Tag)
				isJobInClosed = true
			}
		}

		color.Magenta("Program exit\n")
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
