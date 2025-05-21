package utils

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gopkg.in/yaml.v2"
)

type ProblemRepository = map[int64]models.Problem

func LoadProblems(dir string) (ProblemRepository, error) {
	problems := make(ProblemRepository)

	err := filepath.WalkDir(dir,
		func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("file '%s' read failed, %w\n", path, err)
			}
			var problem models.Problem
			err = yaml.Unmarshal(b, &problem)
			if err != nil {
				return fmt.Errorf("problem unmarshal failed, %w\n", err)
			}
			if _, isExists := problems[problem.ProblemId]; isExists {
				return fmt.Errorf("Duplicate problem id %d\b", problem.ProblemId)
			}

			for i, stage := range problem.Stages {
				for j, _ := range stage.Testcases {
					testcase := &problem.Stages[i].Testcases[j]
					if testcase.SoftTL == 0 {
						testcase.SoftTL = 25
					}
					if testcase.HardTL == 0 {
						testcase.HardTL = 30
					}
				}
			}

			problems[problem.ProblemId] = problem
			return nil
		})

	if err != nil {
		return problems, fmt.Errorf("filepath walk failed, %w\n", err)
	}
	fmt.Fprintf(os.Stderr, "problems: %+v\n", problems)

	return problems, nil
}
