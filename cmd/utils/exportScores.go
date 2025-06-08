package utils

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

type stageScoreWithTime struct {
	Score int64
	Time  int64
}

var exportScoresCmd = &cobra.Command{
	Use:   "export-scores",
	Short: "Export scores to csv file",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(
			viper.GetString("pg-connection-string"))))
		db := bun.NewDB(sqldb, pgdialect.New())

		problems, err := utils.LoadProblems(viper.GetString("problems-dir"))
		if err != nil {
			return fmt.Errorf("read problems failed, %w", err)
		}

		problem, ok := problems[viper.GetInt64("problem-id")]
		if !ok {
			return fmt.Errorf("problem with id '%d' not exist, %w",
				viper.GetInt64("problem-id"), err)
		}

		var lastSubs []models.Submission
		err = db.NewSelect().Model(&lastSubs).
			DistinctOn("csid").
			Column("csid", "score_score", "score_stages").
			Where("problemid = ?", viper.GetInt64("problem-id")).
			OrderExpr("csid, id DESC").
			Scan(ctx)
		if err != nil {
			return fmt.Errorf("get last submissions failed, %w", err)
		}

		// [csId][stage]
		table := make(map[string]map[string]stageScoreWithTime)
		for _, sub := range lastSubs {
			if _, ok := table[sub.CsId]; !ok {
				table[sub.CsId] = make(map[string]stageScoreWithTime)
			}
			for name, score := range sub.Score.Stages {
				table[sub.CsId][name] = stageScoreWithTime{
					Score: score,
				}
			}
		}

		ret := [][]string{[]string{"csid"}}
		// Index
		for name, _ := range problem.Stages {
			ret[0] = append(ret[0], fmt.Sprintf("%s-score", name))
			ret[0] = append(ret[0], fmt.Sprintf("%s-time", name))
		}

		for csId, _ := range table {
			row := []string{csId}
			for name, _ := range problem.Stages {
				row = append(row, strconv.FormatInt(table[csId][name].Score, 10))

				var sub models.Submission
				err := db.NewSelect().Model(&sub).
					Column("csid", "score_stages", "submitat").
					Where("score_stages->? > '0'::jsonb", name).
					Order("id ASC").Limit(1).Scan(ctx)
				if err == nil {
					row = append(row, strconv.FormatInt(sub.SubmitAt, 10))
				} else {
					row = append(row, strconv.FormatInt(0, 10))
				}
			}
			ret = append(ret, row)
		}

		f, err := os.OpenFile(viper.GetString("file"),
			os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			return fmt.Errorf("file open failed, %w", err)
		}
		defer f.Close()

		csvWriter := csv.NewWriter(f)
		csvWriter.WriteAll(ret)

		return nil
	},
}

func init() {
	exportScoresCmd.Flags().
		String("problems-dir", "./problems", "Directory to read problems")
	exportScoresCmd.Flags().
		Int64("problem-id", 1, "Problem id to export")
	exportScoresCmd.Flags().
		String("pg-connection-string", "", "Postgres connection string")
	exportScoresCmd.Flags().
		StringP("file", "f", "export.csv", "Output file")
}
