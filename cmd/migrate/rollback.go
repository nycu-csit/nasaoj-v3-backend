package migrate

import (
	"database/sql"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/migrate"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/migrations"
)

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback from last migration group",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(
			viper.GetString("pg-connection-string"))))
		db := bun.NewDB(sqldb, pgdialect.New())

		migrator := migrate.NewMigrator(db, migrations.Migrations)

		if err := migrator.Lock(ctx); err != nil {
			return err
		}
		defer migrator.Unlock(ctx)

		group, err := migrator.Rollback(ctx)
		if err != nil {
			return err
		}

		if group.IsZero() {
			fmt.Printf("there are no groups to roll back\n")
			return nil
		}

		fmt.Printf("roll back from %s\n", group)
		return nil
	},
}
