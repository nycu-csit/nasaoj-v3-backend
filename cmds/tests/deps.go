package tests

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/implements"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.uber.org/zap"
)

var depsProblemsCmd = &cobra.Command{
	Use:   "problems",
	Short: "Check dependencies on problems",
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
		utils.InitDB(ctx, bunDB)

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

		problems, err := db.GetAllProblems(ctx, nil)
		if err != nil {
			panic(err)
		}

		for problemSHA1, problem := range problems {
			in := make(map[string]int)
			for _, stage := range problem.Stages {
				for _, next := range stage.Before {
					in[next]++
				}
			}

			fmt.Printf("%s (%s)\n", problem.Name, problemSHA1)

			var traverse func(string, int)
			traverse = func(name string, depth int) {
				fmt.Printf("%s%s\n", strings.Repeat("-", depth), name)
				for _, next := range problem.Stages[name].Before {
					traverse(next, depth+1)
				}
			}

			for stageName := range problem.Stages {
				if in[stageName] == 0 {
					traverse(stageName, 1)
				}
			}
		}
	},
}

var depsToolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "Check dependencies on tools",
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
		utils.InitDB(ctx, bunDB)

		db := implements.NewPostgres(bunDB, nil)

		err = utils.LoadProblems(ctx, db, db,
			viper.GetString("tools-dir"),
			viper.GetString("testcases-dir"))
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Panic("failed to load tools",
					zap.Error(err))
			panic(err)
		}
	},
}

var depsCmd = &cobra.Command{
	Use:   "deps",
	Short: "Check dependencies",
}

func init() {
	depsCmd.AddCommand(depsProblemsCmd)
	depsCmd.AddCommand(depsToolsCmd)
}
