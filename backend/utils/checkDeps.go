package utils

import "fmt"

func CheckDeps(problems ProblemRepository) error {
	for _, problem := range problems {

		stageDepCnt := make(map[string]int64)
		for _, stage := range problem.Stages {
			for _, next := range stage.Before {
				stageDepCnt[next] += 1
			}
		}

		pendingStages := []string{}
		for name, _ := range problem.Stages {
			if _, exist := stageDepCnt[name]; !exist {
				pendingStages = append(pendingStages, name)
			}
		}

		stagePassCnt := 0
		for len(pendingStages) > 0 {
			name := pendingStages[0]
			pendingStages = pendingStages[1:]
			stage := problem.Stages[name]

			for _, next := range stage.Before {
				stageDepCnt[next] -= 1
				if stageDepCnt[next] == 0 {
					pendingStages = append(pendingStages, next)
				}
			}

			stagePassCnt += 1
		}

		if stagePassCnt != len(problem.Stages) {
			return fmt.Errorf("problem '%s' contains cycle\n", problem.Name)
		}
	}

	return nil
}
