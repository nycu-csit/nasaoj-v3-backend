package utils

import (
	"context"
	"os"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"go.uber.org/zap"
)

func Rmrf(
	ctx context.Context,
	path string,
) error {
	if err := os.RemoveAll(path); err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to remove files from previous run",
				zap.String("file.path", path),
				zap.Error(err))
		return err
	}
	return nil
}
