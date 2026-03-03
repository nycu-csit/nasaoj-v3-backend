package utils

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/parsers"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"go.uber.org/zap"
	"go.yaml.in/yaml/v3"
)

func LoadProblems(
	ctx context.Context,
	problems interfaces.ProblemStorage,
	contents interfaces.ContentFileStorage,
	problemDir, testcaseDir string,
) error {
	contentFiles, err := LoadTestcases(ctx, testcaseDir)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to load testcases",
				zap.Error(err))
		return err
	}

	err = filepath.WalkDir(problemDir,
		func(cur string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() { // Skip directory
				return nil
			}

			if d.Type()&fs.ModeSymlink != 0 {
				realPath, err := filepath.EvalSymlinks(cur)
				if err != nil {
					return err
				}

				return LoadProblems(ctx, problems, contents, realPath, testcaseDir)
			}

			b, err := os.ReadFile(cur)
			if err != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to read file",
						zap.String("file.path", cur),
						zap.Error(err))
				return err
			}

			var inputProblem parsers.Problem
			if err := yaml.Unmarshal(b, &inputProblem); err != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to marshal problem to yaml",
						zap.Error(err))
				return err
			}

			problem, err := inputProblem.ToModelProblem(ctx, contentFiles)
			if err != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to transform a problem",
						zap.Error(err))
				return err
			}

			if err := CheckDeps(ctx, problem); err != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to check dependency",
						zap.Error(err))
				return err
			}

			if _, err := problems.InsertProblem(ctx, problem); err != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to insert problem",
						zap.Error(err))
				return err
			}

			if err := contents.InsertContentFiles(
				ctx, problem.PreScript.Files); err != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to insert problem prescript",
						zap.Error(err))
				return err
			}

			if err := contents.InsertContentFiles(
				ctx, problem.PostScript.Files); err != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to insert problem postscript",
						zap.Error(err))
				return err
			}

			for _, stage := range problem.Stages {
				if err := contents.InsertContentFiles(
					ctx, stage.PreScript.Files); err != nil {
					tracing.Logger.Ctx(ctx).
						Error("failed to insert stage prescript",
							zap.Error(err))
					return err
				}

				if err := contents.InsertContentFiles(
					ctx, stage.PostScript.Files); err != nil {
					tracing.Logger.Ctx(ctx).
						Error("failed to insert stage postscript",
							zap.Error(err))
					return err
				}

				for _, testcase := range stage.Testcases {
					if err := contents.InsertContentFiles(
						ctx, testcase.Content.Files); err != nil {
						tracing.Logger.Ctx(ctx).
							Error("failed to insert content files",
								zap.Error(err))
						return err
					}
				}
			}

			return nil
		})
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to walk filepath",
				zap.Error(err))
		return err
	}
	return nil
}
