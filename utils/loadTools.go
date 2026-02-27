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

func LoadTools(
	ctx context.Context,
	tools interfaces.ToolStorage,
	contents interfaces.ContentFileStorage,
	toolDir, testcaseDir string,
) error {
	contentFiles, err := LoadTestcases(ctx, testcaseDir)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to load testcases",
				zap.Error(err))
		return err
	}

	err = filepath.WalkDir(toolDir,
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

				return LoadTools(ctx, tools, contents, realPath, testcaseDir)
			}

			b, err := os.ReadFile(cur)
			if err != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to read file",
						zap.String("file.path", cur),
						zap.Error(err))
				return err
			}

			var inputTool parsers.Tool
			if err := yaml.Unmarshal(b, &inputTool); err != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to marshal tool to yaml",
						zap.Error(err))
				return err
			}

			tool, err := inputTool.ToModelTool(ctx, contentFiles)
			if err != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to transform a tool",
						zap.Error(err))
				return err
			}

			if err := tools.InsertTool(ctx, tool); err != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to insert tool",
						zap.Error(err))
				return err
			}

			if err := contents.InsertContentFiles(
				ctx, tool.PreScript.Files); err != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to insert tool prescript",
						zap.Error(err))
				return err
			}

			if err := contents.InsertContentFiles(
				ctx, tool.PostScript.Files); err != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to insert tool postscript",
						zap.Error(err))
				return err
			}

			if err := contents.InsertContentFiles(
				ctx, tool.Testcase.Content.Files); err != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to insert testcase content files",
						zap.Error(err))
				return err
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
