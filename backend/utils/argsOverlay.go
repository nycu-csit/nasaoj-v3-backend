package utils

import "maps"

func ArgsOverlay(
	argsList ...map[string]string,
) map[string]string {

	ret := make(map[string]string)
	for _, args := range argsList {
		maps.Copy(ret, args)
	}

	return ret
}
