package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gitlab.it.cs.nycu.edu.tw/ytshih/na-judge/models"
	"gopkg.in/yaml.v2"
)

func LoadFiles() ([]models.Problem, error) {
	PROBLEMS_DIR := os.Getenv("PROBLEMS_DIR")

	files, err := os.ReadDir(PROBLEMS_DIR)
	if err != nil {
		panic(err)
	}

	var problems []models.Problem
	for _, file := range files {
		b, err := os.ReadFile(filepath.Join(PROBLEMS_DIR, file.Name()))
		if err != nil {
			return []models.Problem{}, fmt.Errorf("file '%s' read failed\n", file.Name())
		}
		var problem models.Problem
		yaml.Unmarshal(b, &problem)
		problems = append(problems, problem)
	}
	return problems, nil
}
