package imports

import (
	"database/sql"
	"fmt"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/implements"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.uber.org/zap"
)

var problemsCmd = &cobra.Command{
	Use:   "problems",
	Short: "Import problems",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		sqldb, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to init sqlite",
					zap.Error(err))
			panic(err)
		}
		bunDB := bun.NewDB(sqldb, sqlitedialect.New())
		if err := utils.InitDB(ctx, bunDB); err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to init db",
					zap.Error(err))
			panic(err)
		}
		db := implements.NewPostgres(bunDB, nil)

		err = utils.LoadProblems(ctx, db, db,
			viper.GetString("problems-dir"),
			viper.GetString("testcases-dir"))
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to load problems",
					zap.Error(err))
			panic(err)
		}

		exportProblems, err := db.GetAllProblems(ctx)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to get all problems",
					zap.Error(err))
			panic(err)
		}

		exportContentFiles, err := db.GetAllContentFiles(ctx)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to get all content files",
					zap.Error(err))
			panic(err)
		}

		var contentFilesPayload []api.PutContentFilesInput
		for _, file := range exportContentFiles {
			contentFilesPayload = append(contentFilesPayload,
				api.PutContentFilesInput{
					SHA1:       file.SHA1,
					Name:       file.Name,
					Payload:    file.Payload,
					IsTemplate: file.IsTemplate,
				})
		}

		importIdMap := make(map[int64]struct{})
		for _, id := range viper.GetIntSlice("problem-id") {
			importIdMap[int64(id)] = struct{}{}
		}
		if len(importIdMap) == 0 {
			for _, problem := range exportProblems {
				importIdMap[problem.ProblemId] = struct{}{}
			}
		}

		var problems []api.PutProblemsInputProblem
		for _, problem := range exportProblems {
			if _, ok := importIdMap[problem.ProblemId]; !ok {
				continue
			}
			// Dehydrate to remove payload from content files
			db.DehydrateProblem(&problem)

			problems = append(problems, api.PutProblemsInputProblem{
				ProblemId:   problem.ProblemId,
				Name:        problem.Name,
				Description: problem.Description,
				Spec:        problem.Spec,
				Stages:      problem.Stages,
				Args:        problem.Args,
				Tag:         problem.Tag,
				PreScript:   problem.PreScript,
				PostScript:  problem.PostScript,
				Cooldown:    problem.Cooldown,
				Deadline:    problem.Deadline,
				ScoreType:   problem.ScoreType,
			})
		}

		var res []api.PutProblemsOutputProblem
		_ = utils.RestyBuilder(ctx, "/api/admin/problems",
			func(r *resty.Request) *resty.Request {
				r.Method = resty.MethodPut
				return r.
					SetBody(problems).
					SetResult(&res)
			})

		for _, problem := range res {
			fmt.Printf("%s\t%d\t%s\n", problem.SHA1, problem.ProblemId, problem.Name)
		}

		_ = utils.RestyBuilder(ctx, "/api/admin/content-files",
			func(r *resty.Request) *resty.Request {
				r.Method = resty.MethodPut
				return r.
					SetBody(contentFilesPayload)
			})

		// io.WriteString(os.Stdout, string(resp.Body()[:]))
	},
}

func init() {
	problemsCmd.Flags().
		String("problems-dir", "./problems", "Directory to read problems")
	problemsCmd.Flags().
		String("testcases-dir", "./testcases", "Directory to read testcases")
	// Use IntSlice because viper doesn't support Int64SliceP
	problemsCmd.Flags().
		IntSliceP("problem-id", "i", []int{}, "Import only specific ids")
}
