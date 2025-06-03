package utils

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

var importUserCmd = &cobra.Command{
	Use:   "import-user",
	Short: "Import user from csv file (line format: {csid},{subnet})",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Initialize DB instance
		sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(
			viper.GetString("pg-connection-string"))))
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

				id, err := strconv.ParseInt(parts[1], 10, 64)
				if err != nil {
					return fmt.Errorf("failed to parse id, %s", err)
				}
				userResource := models.UserResource{
					CsId: parts[0],
					Id:   id,
				}
				_, err = tx.NewInsert().Model(&userResource).
					On("CONFLICT (csid) DO NOTHING").
					Exec(ctx)
				if err != nil {
					return fmt.Errorf("failed to insert user resource: %w", err)
				}
			}

			return nil
		})
	},
}
