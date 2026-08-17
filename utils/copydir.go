package utils

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"go.uber.org/zap"
)

// CopyDir recursively copies the contents of src into dst, creating dst if
// needed and overwriting any files that already exist at the destination
// path. A missing src is not an error - there's simply nothing to copy.
func CopyDir(
	ctx context.Context,
	dst string,
	src string,
) error {
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil
	}

	return filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0700)
		}

		if !d.Type().IsRegular() {
			return nil
		}

		if err := CopyFile(target, p); err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to copy file",
					zap.String("src", p),
					zap.String("dst", target),
					zap.Error(err))
			return err
		}
		return nil
	})
}

// CopyFile copies a single file from src to dst, creating dst's parent
// directory and overwriting dst if it already exists.
func CopyFile(dst string, src string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
