package utils

import "gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"

func CalcScore(
	problem models.Problem,
	sub models.Submission,
) models.Score {
	stageJobCnt := make(map[string]int)
	for _, job := range sub.Jobs {
		if job.Status == models.StatusSuccess {
			stageJobCnt[job.StageName] += 1
		}
	}

	ret := models.Score{
		Score:  0,
		Stages: make(map[string]int64),
	}
	for stageName, stage := range problem.Stages {
		cnt, ok := stageJobCnt[stageName]
		if ok && cnt == len(stage.Testcases) {
			ret.Score += stage.Score
			ret.Stages[stageName] = stage.Score
		}
	}

	return ret
}
