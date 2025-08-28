package imports

import (
	"database/sql"
	"fmt"
	"net/url"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
	apiHandlers "gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/implements"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
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

		var contentFilesPayload []apiHandlers.PutContentFilesInput
		for _, file := range exportContentFiles {
			contentFilesPayload = append(contentFilesPayload,
				apiHandlers.PutContentFilesInput{
					SHA1:       file.SHA1,
					Name:       file.Name,
					Payload:    file.Payload,
					IsTemplate: file.IsTemplate,
				})
		}

		client, ok := ctx.Value(types.RestyClient("")).(*resty.Client)
		if !ok {
			tracing.Logger.Ctx(ctx).
				Panic("resty client not exist")
			panic(types.ContextNotExistError)
		}

		toolsURL, err := url.JoinPath(
			viper.GetString("endpoint"), "/api/admin/tools")
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to construct url",
					zap.Error(err))
			panic(err)
		}
		resp, err := client.R().
			SetBody(exportTools).
			Put(toolsURL)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to put tools",
					zap.String("url", toolsURL),
					zap.Error(err))
			panic(err)
		}

		if !resp.IsSuccess() {
			tracing.Logger.Ctx(ctx).
				Panic("resp.IsSuccess failed")
			panic(types.HTTPRequestFailedError)
		}
		fmt.Printf("%v\n", string(resp.Body()[:]))

		contentFilesURL, err := url.JoinPath(
			viper.GetString("endpoint"), "/api/admin/content-files")
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to construct url",
					zap.Error(err))
			panic(err)
		}
		resp, err = client.R().
			SetBody(contentFilesPayload).
			Put(contentFilesURL)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to put content files",
					zap.String("url", contentFilesURL),
					zap.Error(err))
			panic(err)
		}

		if !resp.IsSuccess() {
			tracing.Logger.Ctx(ctx).
				Panic("resp.IsSuccess failed")
			panic(types.HTTPRequestFailedError)
		}
		fmt.Printf("%v\n", string(resp.Body()[:]))
	},
}

func init() {
	toolsCmd.Flags().
		String("tools-dir", "./tools", "Directory to read tools")
	toolsCmd.Flags().
		String("testcases-dir", "./testcases", "Directory to read testcases")
}
