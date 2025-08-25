package utils

import (
	"context"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"go.uber.org/zap"
)

func CheckDeps(
	ctx context.Context,
	problem models.Problem,
) error {
	tracing.Logger.Ctx(ctx).
		Info("checking problem",
			zap.String("problem.name", problem.Name))

	stageDepCnt := make(map[string]int64)
	for _, stage := range problem.Stages {
		for _, next := range stage.Before {
			if _, exist := problem.Stages[next]; !exist {
				tracing.Logger.Ctx(ctx).
					Error("dependency not exist",
						zap.String("problem.name", problem.Name),
						zap.String("stage.name", next))
				return types.StageNotExistError
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
		tracing.Logger.Ctx(ctx).
			Error("problem contains cycle",
				zap.String("problem.name", problem.Name))
		return types.ProblemContainsCycleError
	}

	return nil
}
