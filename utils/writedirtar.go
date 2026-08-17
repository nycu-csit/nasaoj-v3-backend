package utils

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"go.uber.org/zap"
)

// WriteDirTar archives every regular file under basepath whose path relative
// to basepath matches at least one of patterns (filepath.Match syntax - no
// "**"; a pattern containing "/" matches that literal path structure). It
// returns the relative paths that were archived, or (nil, nil) when nothing
// matched.
func WriteDirTar(
	ctx context.Context,
	tarbuf *bytes.Buffer,
	basepath string,
	patterns []string,
) ([]string, error) {
	if len(patterns) == 0 {
		return nil, nil
	}

	var matched []string
	tw := tar.NewWriter(tarbuf)

	err := filepath.WalkDir(basepath, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(basepath, p)
		if err != nil {
			return err
		}
		if !matchesAny(rel, patterns) {
			return nil
		}

		f, err := os.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()

		info, err := f.Stat()
		if err != nil {
			return err
		}

		if err := tw.WriteHeader(&tar.Header{
			Name: rel,
			Mode: int64(info.Mode().Perm()),
			Size: info.Size(),
		}); err != nil {
			return err
		}
		if _, err := io.Copy(tw, f); err != nil {
			return err
		}

		matched = append(matched, rel)
		return nil
	})
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to write dir tar",
				zap.String("basepath", basepath),
				zap.Error(err))
		return nil, err
	}

	if err := tw.Close(); err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to close tar writer",
				zap.Error(err))
		return nil, err
	}

	return matched, nil
}

func matchesAny(rel string, patterns []string) bool {
	for _, pattern := range patterns {
		if ok, _ := filepath.Match(pattern, rel); ok {
			return true
		}
	}
	return false
}
