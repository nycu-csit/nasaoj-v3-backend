package imports

import (
	"database/sql"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/implements"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

var problemsCmd = &cobra.Command{
	Use:   "problems",
	Short: "Import problems",
	Run: func(cmd *cobra.Command, args []string) {
		sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(
			viper.GetString("pg-connection-string"))))
		bunDB := bun.NewDB(sqldb, pgdialect.New())
		db := implements.NewBunDatabase(bunDB)
		problems := implements.NewDBProblemStorage(db)

		err := utils.LoadProblems(cmd.Context(), problems,
			viper.GetString("problems-dir"), viper.GetString("testcases-dir"))
		if err != nil {
			panic(err)
		}
	},
}

func init() {
	problemsCmd.Flags().
		String("problems-dir", "./problems", "Directory to read problems")
	problemsCmd.Flags().
		String("testcases-dir", "./testcases", "Directory to read testcases")
	problemsCmd.Flags().
		String("pg-connection-string",
			"postgres://poop:poop@pg:5432/poop?sslmode=disable",
			"Postgres connection string")
}
