package imports

import (
	"database/sql"
	"io"
	"os"

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

		var problems []api.PutProblemsInputProblem
		for sha1, problem := range exportProblems {
			// Dehydrate to remove payload from content files
			db.DehydrateProblem(&problem)

			problems = append(problems, api.PutProblemsInputProblem{
				SHA1: sha1,

				ProblemId:   problem.ProblemId,
				Name:        problem.Name,
				Description: problem.Description,
				PdfUrl:      problem.PdfUrl,
				Stages:      problem.Stages,
				Args:        problem.Args,
				Tag:         problem.Tag,
				PreScript:   problem.PreScript,
				PostScript:  problem.PostScript,
				Cooldown:    problem.Cooldown,
				Deadline:    problem.Deadline,
			})
		}
		resp := utils.RestyBuilder(ctx, "/api/admin/problems",
			func(r *resty.Request) *resty.Request {
				r.Method = resty.MethodPut
				return r.
					SetBody(problems)
			})

		io.WriteString(os.Stdout, string(resp.Body()[:]))

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
}
