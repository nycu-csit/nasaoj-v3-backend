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

// LastImportedTools stores the result of the last import for programmatic access
var LastImportedTools []api.PutToolsOutputTool

var ToolsCmd = &cobra.Command{
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
		db := implements.NewPostgres(bunDB, nil)

		err = utils.LoadTools(ctx, db, db,
			viper.GetString("tools-dir"),
			viper.GetString("testcases-dir"))
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to load tools",
					zap.Error(err))
			panic(err)
		}

		exportTools, err := db.GetAllTools(ctx, nil)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to get all tools",
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
		for _, id := range viper.GetIntSlice("tool-id") {
			importIdMap[int64(id)] = struct{}{}
		}
		if len(importIdMap) == 0 {
			for _, tool := range exportTools {
				importIdMap[tool.ToolId] = struct{}{}
			}
		}

		var tools []api.PutToolsInputTool
		for _, tool := range exportTools {
			if _, ok := importIdMap[tool.ToolId]; !ok {
				continue
			}
			// Dehydrate to remove payload from content files
			db.DehydrateTool(&tool)

			tools = append(tools, api.PutToolsInputTool{
				ToolId:      tool.ToolId,
				Name:        tool.Name,
				Description: tool.Description,
				Testcase:    tool.Testcase,
				Args:        tool.Args,
				InputArgs:   tool.InputArgs,
				Tag:         tool.Tag,
				PreScript:   tool.PreScript,
				PostScript:  tool.PostScript,
				Cooldown:    tool.Cooldown,
			})
		}

		LastImportedTools = nil
		_ = utils.RestyBuilder(ctx, "/api/admin/tools",
			func(r *resty.Request) *resty.Request {
				r.Method = resty.MethodPut
				return r.
					SetBody(tools).
					SetResult(&LastImportedTools)
			})

		for _, tool := range LastImportedTools {
			fmt.Printf("%s\t%d\t%s\n", tool.SHA1, tool.ToolId, tool.Name)
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
	ToolsCmd.Flags().
		String("tools-dir", "./tools", "Directory to read tools")
	ToolsCmd.Flags().
		String("testcases-dir", "./testcases", "Directory to read testcases")
		// Use IntSlice because viper doesn't support Int64SliceP
	ToolsCmd.Flags().
		IntSliceP("tool-id", "i", []int{}, "Import only specific ids")
}
