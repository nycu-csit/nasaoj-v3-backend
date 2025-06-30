package utils

import (
	"bufio"
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
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/implements"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

var importUserCmd = &cobra.Command{
	Use:   "import-user",
	Short: "Import user from csv file (line format: {csid},{roomid},{subnet})",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Initialize DB instance
		sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(
			viper.GetString("pg-connection-string"))))
		bunDB := bun.NewDB(sqldb, pgdialect.New())
		db := implements.NewBunDatabase(bunDB)

		csvFilePath := args[0]
		file, err := os.Open(csvFilePath)
		if err != nil {
			return fmt.Errorf("open file failed, %w", err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		var userResources []models.UserResource
		for scanner.Scan() {
			line := scanner.Text()
			parts := strings.Split(line, ",")
			if len(parts) != 3 {
				continue
			}

			id, err := strconv.ParseInt(parts[2], 10, 64)
			if err != nil {
				return fmt.Errorf("parse id failed '%s', %w", parts[2], err)
			}

			userResources = append(userResources, models.UserResource{
				CsId:   parts[0],
				RoomId: parts[1],
				Id:     id,
			})
		}

		if err := bunDB.ResetModel(ctx, (*models.UserResource)(nil)); err != nil {
			return fmt.Errorf("reset user resource failed, %w", err)
		}

		err = db.InsertUserResources(ctx, userResources)
		if err != nil {
			return fmt.Errorf("insert user resource failed, %w", err)
		}

		return nil
	},
}
