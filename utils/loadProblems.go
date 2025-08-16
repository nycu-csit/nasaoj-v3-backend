package utils

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/parsers"
	"gopkg.in/yaml.v3"
)

func LoadProblems(
	ctx context.Context,
	problems interfaces.ProblemStorage,
	problemDir, testcaseDir string,
) error {
	contentFiles, err := LoadTestcases(testcaseDir)
	if err != nil {
		return fmt.Errorf("testcase load failed, %w", err)
	}

	err = filepath.WalkDir(problemDir,
		func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() { // Skip directory
				return nil
			}

			b, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("file '%s' read failed, %w", path, err)
			}

			var inputProblem parsers.Problem
			if err := yaml.Unmarshal(b, &inputProblem); err != nil {
				return fmt.Errorf("problem unmarshal failed, %w", err)
			}

			problem, err := inputProblem.ToModelProblem(contentFiles)
			if err != nil {
				return fmt.Errorf("problem transform failed, %w", err)
			}

			if err := CheckDeps(problem); err != nil {
				return fmt.Errorf("check dependency failed, %w", err)
			}

			if err := problems.InsertProblem(ctx, problem); err != nil {
				return fmt.Errorf("problem insertion failed, %w", err)
			}

			if err := problems.InsertContentFiles(
				ctx, problem.PreScript.Files); err != nil {
				return fmt.Errorf("problem prescript insert failed, %w", err)
			}

			if err := problems.InsertContentFiles(
				ctx, problem.PostScript.Files); err != nil {
				return fmt.Errorf("problem postscript insert failed, %w", err)
			}

			for _, stage := range problem.Stages {
				if err := problems.InsertContentFiles(
					ctx, stage.PreScript.Files); err != nil {
					return fmt.Errorf("stage prescript insert failed, %w", err)
				}

				if err := problems.InsertContentFiles(
					ctx, stage.PostScript.Files); err != nil {
					return fmt.Errorf("stage postscript insert failed, %w", err)
				}

				for _, testcase := range stage.Testcases {
					if err := problems.InsertContentFiles(
						ctx, testcase.Content.Files); err != nil {
						return fmt.Errorf("content files insert failed, %w", err)
					}
				}
			}

			return nil
		})
	if err != nil {
		return fmt.Errorf("filepath walk failed, %w", err)
	}
	return nil
}
