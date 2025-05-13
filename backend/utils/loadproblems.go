package utils

import (
	"fmt"
	"os"
	"path/filepath"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gopkg.in/yaml.v2"
)

type ProblemRepository = map[int64]models.Problem

func LoadProblems(dir string) (ProblemRepository, error) {
	problems := make(ProblemRepository)

	files, err := os.ReadDir(dir)
	if err != nil {
		return problems, fmt.Errorf("ReadDir failed, %w\n", err)
	}

	for _, file := range files {
		b, err := os.ReadFile(filepath.Join(dir, file.Name()))
		if err != nil {
			return problems, fmt.Errorf("File '%s' read failed\n", file.Name())
		}
		var problem models.Problem
		if err != nil {
			return problems, fmt.Errorf("Problem default value set failed, %w\n", err)
		}
		err = yaml.Unmarshal(b, &problem)
		if err != nil {
			return problems, fmt.Errorf("Problem unmarshal failed, %w\n", err)
		}
		if _, isExists := problems[problem.ProblemId]; isExists {
			return problems, fmt.Errorf("Duplicate problem id %d\n", problem.ProblemId)
		}

		for i, stage := range problem.Stages {
			for j, _ := range stage.Testcases {
				softTL := &problem.Stages[i].Testcases[j].SoftTL
				hardTL := &problem.Stages[i].Testcases[j].HardTL
				if *softTL == 0 {
					*softTL = 25
				}
				if *hardTL == 0 {
					*hardTL = 30
				}
			}
		}

		problems[problem.ProblemId] = problem
	}

	fmt.Fprintf(os.Stderr, "problems: %+v\n", problems)

	return problems, nil
}
