package cmd

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

var ImportUserCmd = &cobra.Command{
	Use:   "import-user",
	Short: "Import user from csv file (line format: {csid},{subnet})",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// Initialize DB instance
		dsn := os.Getenv("PG_CONNECTION_STRING")

		sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
		db := bun.NewDB(sqldb, pgdialect.New())

		csvFilePath := args[0]
		file, err := os.Open(csvFilePath)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := scanner.Text()
				parts := strings.Split(line, ",")
				if len(parts) != 2 {
					return fmt.Errorf("invalid line format: %s", line)
				}
				csId := parts[0]
				subnet := parts[1]

				resource := models.Resource{
					Subnet: subnet,
				}
				_, err := tx.NewInsert().Model(&resource).On("CONFLICT (subnet) DO NOTHING").Exec(ctx)
				if err != nil {
					return fmt.Errorf("failed to insert resource: %w", err)
				}

				userResource := models.UserResource{
					CsId:   csId,
					Subnet: subnet,
				}
				_, err = tx.NewInsert().Model(&userResource).On("CONFLICT (csid) DO NOTHING").On("CONFLICT (subnet) DO NOTHING").Exec(ctx)
				if err != nil {
					return fmt.Errorf("failed to insert user resource: %w", err)
				}
			}

			return nil
		})
	},
}
