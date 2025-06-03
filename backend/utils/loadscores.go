package utils

import (
	"context"
	"net/http"

	"github.com/uptrace/bun"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type ScoreTable = map[string]models.UserScore

func LoadScores(
	ctx context.Context,
	db *bun.DB,
	problems ProblemRepository,
) (map[int64]ScoreTable, error) {
	ret := make(map[int64]ScoreTable)

	for _, problem := range problems {
		problemId := problem.ProblemId
		ret[problemId] = make(ScoreTable)

		var submissions []models.Submission
		err := db.NewSelect().DistinctOn("csid").Model(&submissions).
			Where("problemid = ?", problemId).
			Relation("Jobs").Order("csid, id DESC").Scan(ctx)
		if err != nil {
			return ret, middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "submission query failed",
				OriginError: err,
			}
		}

		for _, submission := range submissions {
			ret[problemId][submission.CsId] = submission.Score
		}
	}
	return ret, nil
}
