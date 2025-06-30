package migrates

import (
	"database/sql"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/migrate"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/migrations"
)

var migrator *migrate.Migrator

var RootCmd = &cobra.Command{
	Use:   "migrate",
	Short: "NASAOJ - Migration related commands",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(
			viper.GetString("pg-connection-string"))))
		db := bun.NewDB(sqldb, pgdialect.New())

		migrator = migrate.NewMigrator(db, migrations.Migrations)
	},
}

func init() {
	RootCmd.AddCommand(upCmd)
	RootCmd.AddCommand(initCmd)
	RootCmd.AddCommand(rollbackCmd)
	RootCmd.AddCommand(lockCmd)
	RootCmd.AddCommand(unlockCmd)
	RootCmd.PersistentFlags().
		String("pg-connection-string",
			"postgres://poop:poop@pg:5432/poop?sslmode=disable",
			"Postgres connection string")
}
