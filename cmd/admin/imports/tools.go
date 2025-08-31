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
	api "gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/implements"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.uber.org/zap"
)

var toolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "Import tools",
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
		db := implements.NewBunDatabase(bunDB)
		tools := implements.NewDBToolStorage(db)

		err = utils.LoadTools(ctx, tools,
			viper.GetString("tools-dir"),
			viper.GetString("testcases-dir"))
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to load tools",
					zap.Error(err))
			panic(err)
		}

		exportTools, err := tools.GetAllTools(ctx)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to get all tools",
					zap.Error(err))
			panic(err)
		}

		for sha1, tool := range exportTools {
			exportTools[sha1] = tool
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

		resp := utils.RestyBuilder(cmd, "/api/admin/tools",
			func(r *resty.Request) *resty.Request {
				r.Method = resty.MethodPut
				return r.
					SetBody(exportTools)
			})

		io.WriteString(os.Stdout, string(resp.Body()[:]))

		resp = utils.RestyBuilder(cmd, "/api/admin/content-files",
			func(r *resty.Request) *resty.Request {
				r.Method = resty.MethodPut
				return r.
					SetBody(contentFilesPayload)
			})

		io.WriteString(os.Stdout, string(resp.Body()[:]))
	},
}

func init() {
	toolsCmd.Flags().
		String("tools-dir", "./tools", "Directory to read tools")
	toolsCmd.Flags().
		String("testcases-dir", "./testcases", "Directory to read testcases")
}
