package utils

import (
	"context"
	"fmt"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type ScoreTable = map[string]models.Score

func LoadScores(
	ctx context.Context,
	db interfaces.Database,
	problems ProblemRepository,
) (map[int64]ScoreTable, error) {
	ret := make(map[int64]ScoreTable)

	for _, problem := range problems {
		problemId := problem.ProblemId
		ret[problemId] = make(ScoreTable)

		submissions, err := db.GetAllSubmission(ctx, problemId)
		if err != nil {
			return ret, fmt.Errorf("get all submission failed, %w", err)
		}

		for _, submission := range submissions {
			ret[problemId][submission.CsId] = submission.Score
		}
	}
	return ret, nil
}
