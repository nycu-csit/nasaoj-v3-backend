package utils

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"text/template"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"go.uber.org/zap"
)

func LoadTestcases(
	ctx context.Context,
	dir string,
) (map[string]models.ContentFile, error) {
	ret := make(map[string]models.ContentFile)

	err := filepath.WalkDir(dir,
		func(cur string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			name, err := filepath.Rel(dir, cur)
			if err != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to parse relative path",
						zap.Error(err))
				return err
			}

			if d.Type()&fs.ModeSymlink != 0 {
				realPath, err := filepath.EvalSymlinks(cur)
				if err != nil {
					return err
				}

				m, err := LoadTestcases(ctx, realPath)
				if err != nil {
					return err
				}

				for subPath, file := range m {
					ret[path.Join(name, subPath)] = file
				}

				return nil
			}

			b, err := os.ReadFile(cur)
			if err != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to read file",
						zap.String("file.path", cur),
						zap.Error(err))
				return err
			}

			tracing.Logger.Ctx(ctx).
				Info("found file",
					zap.String("file.name", name))

			h := sha1.Sum(b)
			contentFile := models.ContentFile{
				Name:    name,
				SHA1:    hex.EncodeToString(h[:]),
				Payload: string(b[:]),
			}
			if path.Ext(name) == ".tmpl" {
				tmpl, err := template.New(name).Parse(string(b[:]))
				if err != nil {
					tracing.Logger.Ctx(ctx).
						Error("failed to template file",
							zap.Error(err))
					return err
				}
				contentFile.IsTemplate = true
				contentFile.Template = tmpl
			} else {
				contentFile.IsTemplate = false
				contentFile.Template = nil
			}
			ret[name] = contentFile

			tracing.Logger.Ctx(ctx).
				Info("load testcase",
					zap.String("testcase.name", name))
			return nil
		})
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to walk filepath",
				zap.Error(err))
		return map[string]models.ContentFile{}, nil
	}

	return ret, nil
}
