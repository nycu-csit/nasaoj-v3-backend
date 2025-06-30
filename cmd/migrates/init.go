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

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize migration table",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(
			viper.GetString("pg-connection-string"))))
		db := bun.NewDB(sqldb, pgdialect.New())

		migrator := migrate.NewMigrator(db, migrations.Migrations)
		return migrator.Init(ctx)
	},
}
