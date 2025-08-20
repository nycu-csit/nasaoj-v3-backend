package tests

import (
	"database/sql"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/implements"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

var problemsCmd = &cobra.Command{
	Use:   "problems",
	Short: "Check dependencies on problems",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		sqldb, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
		if err != nil {
			return fmt.Errorf("sqlite init failed, %w", err)
		}
		bunDB := bun.NewDB(sqldb, sqlitedialect.New())
		utils.InitDB(ctx, bunDB)

		db := implements.NewBunDatabase(bunDB)
		problems := implements.NewDBProblemStorage(db)

		return utils.LoadProblems(ctx, problems,
			viper.GetString("problems-dir"),
			viper.GetString("testcases-dir"))
	},
}

var toolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "Check dependencies on tools",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		sqldb, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
		if err != nil {
			return fmt.Errorf("sqlite init failed, %w", err)
		}
		bunDB := bun.NewDB(sqldb, sqlitedialect.New())
		utils.InitDB(ctx, bunDB)

		db := implements.NewBunDatabase(bunDB)
		tools := implements.NewDBProblemStorage(db)

		return utils.LoadProblems(ctx, tools,
			viper.GetString("tools-dir"),
			viper.GetString("testcases-dir"))
	},
}

var depsCmd = &cobra.Command{
	Use:   "deps",
	Short: "Check dependencies",
}

func init() {
	depsCmd.AddCommand(problemsCmd)
	depsCmd.AddCommand(toolsCmd)
}
