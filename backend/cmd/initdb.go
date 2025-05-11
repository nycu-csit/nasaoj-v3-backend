package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/utils"
)

var InitDBCmd = &cobra.Command{
	Use:   "initdb",
	Short: "Initialize and reset the database (NOTE: all data will be lost)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// Initialize DB instance
		dsn := fmt.Sprintf("postgres://%s:%s@%s:5432/%s?sslmode=disable",
			os.Getenv("PG_USER"), os.Getenv("PG_PASS"),
			os.Getenv("PG_HOST"), os.Getenv("PG_DB"))

		sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
		db := bun.NewDB(sqldb, pgdialect.New())

		return utils.InitDB(ctx, db)
	},
}
