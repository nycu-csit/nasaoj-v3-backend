package utils

import (
	"context"
	"database/sql"
	"os"

	"github.com/spf13/cobra"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/utils"
)

var initDBCmd = &cobra.Command{
	Use:   "initdb",
	Short: "Initialize and reset the database (NOTE: all data will be lost)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// Initialize DB instance
		dsn := os.Getenv("PG_CONNECTION_STRING")

		sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
		db := bun.NewDB(sqldb, pgdialect.New())

		return utils.InitDB(ctx, db)
	},
}
