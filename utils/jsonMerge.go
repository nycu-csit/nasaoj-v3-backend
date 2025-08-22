package utils

import (
	"encoding/json"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch/v5"
)

func JSONMerge(
	base interface{},
	patches ...interface{},
) ([]byte, error) {
	ret, err := json.Marshal(base)
	if err != nil {
		return []byte{}, fmt.Errorf("json marshal failed, %w", err)
	}

	for _, patch := range patches {
		b, err := json.Marshal(patch)
		if err != nil {
			return []byte{}, fmt.Errorf("json marshal failed, %w", err)
		}

		ret, err = jsonpatch.MergePatch(ret, b)
		if err != nil {
			return []byte{}, fmt.Errorf("json merge failed, %w", err)
		}
	}

	return ret, nil
}
