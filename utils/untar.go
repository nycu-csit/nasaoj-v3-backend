package utils

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"os"
	"path"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"go.uber.org/zap"
)

func UnTar(
	ctx context.Context,
	basepath string,
	tarbuf []byte,
) error {
	if err := os.RemoveAll(basepath); err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to remove files from previous run",
				zap.String("file.path", basepath),
				zap.Error(err))
		return err
	}

	if err := os.MkdirAll(basepath, 0700); err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to make directory",
				zap.String("file.path", basepath),
				zap.Error(err))
		return err
	}

	buf := bytes.NewBuffer(tarbuf)
	tr := tar.NewReader(buf)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		b, err := io.ReadAll(tr)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to read file from tar",
					zap.Error(err))
			return err
		}

		filepath := path.Join(basepath, hdr.Name)
		err = os.MkdirAll(path.Dir(filepath), 0700)
		if err != nil && !os.IsExist(err) {
			tracing.Logger.Ctx(ctx).
				Error("failed to make directory for file",
					zap.String("file.path", filepath),
					zap.Error(err))
			return err
		}

		if hdr.Typeflag != tar.TypeReg {
			continue
		}

		if err := os.WriteFile(filepath, b, os.FileMode(hdr.Mode)); err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to write file",
					zap.String("file.path", filepath),
					zap.Error(err))
			return err
		}
	}

	return nil
}
