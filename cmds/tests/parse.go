package tests

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/implements"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

var parseProblemsCmd = &cobra.Command{
	Use:   "problems",
	Short: "Parse and show problems",
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

		problems, err := db.GetAllProblems(ctx)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to get all problems",
					zap.Error(err))
			panic(err)
		}

		filterId := viper.GetInt64("filter-id")
		for sha1, problem := range problems {
			if filterId != 0 && problem.ProblemId != filterId {
				continue
			}
			fmt.Fprintf(os.Stdout, "---\n# SHA1: %s\n", sha1)
			b, err := yaml.Marshal(problem)
			if err != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to marshal problem",
						zap.Error(err))
				continue
			}
			os.Stdout.Write(b)
			fmt.Fprintln(os.Stdout)
		}
	},
}

var parseToolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "Parse and show tools",
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

		tools, err := db.GetAllTools(ctx)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to get all tools",
					zap.Error(err))
			panic(err)
		}

		filterId := viper.GetInt64("filter-id")
		for sha1, tool := range tools {
			if filterId != 0 && tool.ToolId != filterId {
				continue
			}
			fmt.Fprintf(os.Stdout, "---\n# SHA1: %s\n", sha1)
			b, err := yaml.Marshal(tool)
			if err != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to marshal tool",
						zap.Error(err))
				continue
			}
			os.Stdout.Write(b)
			fmt.Fprintln(os.Stdout)
		}
	},
}

var parseCmd = &cobra.Command{
	Use:   "parse",
	Short: "Parse yaml",
}

func init() {
	parseCmd.PersistentFlags().
		Int64("filter-id", 0, "Filter output by problem/tool ID (0 = no filter)")

	parseCmd.AddCommand(parseProblemsCmd)
	parseCmd.AddCommand(parseToolsCmd)
}
