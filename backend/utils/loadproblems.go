package utils

import (
	"fmt"
	"os"
	"path/filepath"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gopkg.in/yaml.v2"
)

func LoadProblems(dir string) (map[string]models.Problem, error) {
	problems := make(map[string]models.Problem)

	files, err := os.ReadDir(dir)
	if err != nil {
		return problems, fmt.Errorf("ReadDir failed, %s\n", err)
	}

	for _, file := range files {
		b, err := os.ReadFile(filepath.Join(dir, file.Name()))
		if err != nil {
			return problems, fmt.Errorf("File '%s' read failed\n", file.Name())
		}
		var problem models.Problem
		yaml.Unmarshal(b, &problem)
		problems[problem.Name] = problem
	}

	fmt.Fprintf(os.Stderr, "problems: %+v\n", problems)

	return problems, nil
}
