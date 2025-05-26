package utils

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gopkg.in/yaml.v3"
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
				return fmt.Errorf("duplicate problem id %d\b", problem.ProblemId)
			}

			problems[problem.ProblemId] = problem
			return nil
		})
	if err != nil {
		return problems, fmt.Errorf("filepath walk failed, %w\n", err)
	}

	return problems, nil
}
