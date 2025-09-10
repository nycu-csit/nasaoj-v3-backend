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

var stageCmd = &cobra.Command{
	Use:   "stage",
	Short: "Run a single stage on local",
	Run: func(cmd *cobra.Command, args []string) {
		color.Blue("Start running stage '%s' in problem '%d'\n",
			viper.GetString("stage"),
			viper.GetInt64("problem-id"))

		ctx := cmd.Context()

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
		utils.InitDB(ctx, bunDB)

		db := implements.NewBunDatabase(bunDB, implements.DatabaseTypeSqlite)
		jobQue := implements.NewChanJobQueue()
		s3 := implements.NewMapObjectStorage()
		problems := implements.NewDBProblemStorage(bunDB, nil)
		contents := implements.NewDBContentFileStorage(bunDB, nil)
		tokens := implements.NewMapTokenStorage()

		err = utils.LoadProblems(ctx, problems, contents,
			viper.GetString("problems-dir"),
			viper.GetString("testcases-dir"))
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to load problems",
					zap.Error(err))
			panic(err)
		}

		_, err = bunDB.NewUpdate().
			Model((*models.ProblemMarshaled)(nil)).
			Set("is_enabled = ?", true).
			Where("is_enabled = ?", false).
			Exec(ctx)
		if err != nil {
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

		err = workflows.RunStage(ctx, jobQue, db, s3, problems, contents, tokens,
			models.Submission{
				Id:          114514,
				ProblemId:   problem.ProblemId,
				ProblemSHA1: problemMarshaled.SHA1,
				Args:        mockArgs,
			},
			viper.GetString("stage"),
		)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to schedule stage",
					zap.Error(err))
			panic(err)
		}

		color.Cyan("run stage complete\n")

		for {
			if jobQue.Size(problem.Tag) == 0 {
				close(jobIn)
				break
			}
			job, err := jobQue.GetJob(ctx, problem.Tag)
			if err != nil {
				color.Red("get job from job queue failed, %v\n", err)
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

		for {
			_, ok := <-jobDone
			if !ok {
				break // channel closed
			}
		}

		color.Magenta("Program exit\n")
	},
}

func init() {
	stageCmd.Flags().
		String("running-dir", "/tmp/nasaoj-worker", "Directory to run worker")

	stageCmd.Flags().
		Int64("problem-id", 1, "Problem SHA1 to test")
	stageCmd.MarkFlagRequired("problem-id")
	stageCmd.Flags().
		String("stage", "", "The stage to test")
	stageCmd.MarkFlagRequired("stage")
	stageCmd.Flags().
		Int64("buffer-size", 100, "Channel buffer size")
	stageCmd.Flags().
		StringSliceP("arg", "a", []string{}, "Mock submission args (key=value)")
}
