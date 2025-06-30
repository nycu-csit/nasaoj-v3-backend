package utils

import (
	"fmt"
	"os"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

func CheckDeps(problem models.Problem) error {
	fmt.Fprintf(os.Stderr, "Checking problem '%s'\n", problem.Name)

	stageDepCnt := make(map[string]int64)
	for _, stage := range problem.Stages {
		for _, next := range stage.Before {
			if _, exist := problem.Stages[next]; !exist {
				return fmt.Errorf(
					"dependency '%s' in problem '%s' does not exist",
					next, problem.Name)
			}
			stageDepCnt[next] += 1
		}
	}

	pendingStages := []string{}
	for name, _ := range problem.Stages {
		if _, exist := stageDepCnt[name]; !exist {
			pendingStages = append(pendingStages, name)
		}
	}

	stagePass := []string{}
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

		stagePass = append(stagePass, name)
	}

	if len(stagePass) != len(problem.Stages) {
		return fmt.Errorf("problem '%s' contains cycle\n", problem.Name)
	}

	return nil
}
