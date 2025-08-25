package utils

import (
	"context"
	"encoding/json"

	jsonpatch "github.com/evanphx/json-patch/v5"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"go.uber.org/zap"
)

func JSONMerge(
	ctx context.Context,
	base interface{},
	patches ...interface{},
) ([]byte, error) {
	ret, err := json.Marshal(base)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to marshal json",
				zap.Error(err))
		return []byte{}, err
	}

	for _, patch := range patches {
		b, err := json.Marshal(patch)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to marshal json",
					zap.Error(err))
			return []byte{}, err
		}

		ret, err = jsonpatch.MergePatch(ret, b)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to marshal json",
					zap.Error(err))
			return []byte{}, err
		}
	}

	return ret, nil
}
