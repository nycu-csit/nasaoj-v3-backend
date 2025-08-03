package tests

import (
	"database/sql"
	"fmt"
	"strings"

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

var stageCmd = &cobra.Command{
	Use:   "stage",
	Short: "Run a single stage on local",
	RunE: func(cmd *cobra.Command, args []string) error {
		color.Blue("Start running stage '%s' in problem '%d'\n",
			viper.GetString("stage"),
			viper.GetInt64("problem-id"))

		ctx := cmd.Context()

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
		problems := implements.NewDBProblemStorage(db)

		err = utils.LoadProblems(ctx, problems,
			viper.GetString("problems-dir"),
			viper.GetString("testcases-dir"))
		if err != nil {
			return fmt.Errorf("loca problems failed, %w", err)
		}

		problem, err := problems.GetProblem(ctx, viper.GetString("problem-sha1"))

		jobIn := make(chan models.JobWithFile, viper.GetInt64("buffer-size"))
		jobOut := make(chan models.JobWithFile, viper.GetInt64("buffer-size"))
		jobDone := make(chan models.JobWithFile, viper.GetInt64("buffer-size"))

		go workflows.WorkerThread(ctx, jobIn, jobOut)
		go workflows.OutputThread(ctx, jobOut, jobDone)

		err = workflows.RunStage(ctx, jobQue, db, s3, problems,
			models.Submission{
				Id:        114514,
				ProblemId: problem.ProblemId,
				Args:      mockArgs,
			},
			viper.GetString("stage"),
		)
		if err != nil {
			return fmt.Errorf("stage scheduling failed, %w", err)
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
		return nil
	},
}

func init() {
	stageCmd.Flags().
		String("running-dir", "/tmp/nasaoj-worker", "Directory to run worker")

	stageCmd.Flags().
		String("problem-sha1", "", "Problem SHA1 to test")
	stageCmd.MarkFlagRequired("problem-id")
	stageCmd.Flags().
		String("stage", "", "The stage to test")
	stageCmd.MarkFlagRequired("stage")
	stageCmd.Flags().
		Int64("buffer-size", 100, "Channel buffer size")
	stageCmd.Flags().
		StringSliceP("arg", "a", []string{}, "Mock submission args (key=value)")
}
