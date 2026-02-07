package utils

import "gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"

func CalcScore(
	problem models.Problem,
	sub models.Submission,
) (models.Score, bool) {
	isFinal := true
	stageJobCnt := make(map[string]int)
	for _, job := range sub.Jobs {
		switch job.Status {
		case
			models.StatusPending,
			models.StatusRunning,
			models.StatusBlocked:
			isFinal = false
		case models.StatusSuccess:
			stageJobCnt[job.StageName] += 1
		}
	}

	ret := models.Score{
		Score:  0,
		Stages: make(map[string]int64),
	}

	stageIn := make(map[string]int)
	for _, stage := range problem.Stages {
		for _, next := range stage.Before {
			stageIn[next] += 1
		}
	}

	var queue []string
	for stageName := range problem.Stages {
		if cnt, ok := stageIn[stageName]; !ok || cnt == 0 {
			queue = append(queue, stageName)
		}
	}

	for len(queue) > 0 {
		stageName := queue[0]
		queue = queue[1:]
		stage := problem.Stages[stageName]

		cnt, ok := stageJobCnt[stageName]
		if !ok {
			cnt = 0
		}
		if cnt == len(stage.Testcases) {
			ret.Score += stage.Score
			ret.Stages[stageName] = stage.Score

			for _, next := range stage.Before {
				stageIn[next] -= 1
				if stageIn[next] <= 0 {
					queue = append(queue, next)
				}
			}
		}
	}

	return ret, isFinal
}
