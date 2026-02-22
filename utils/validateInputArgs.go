package utils

import (
	"fmt"
	"strconv"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
)

func ValidateInputArgs(
	toolInputArgs []models.ToolInputArg,
	inputArgs map[string]string,
) error {
	for _, arg := range toolInputArgs {
		val, exists := inputArgs[arg.Name]
		if !exists {
			return fmt.Errorf("missing input argument: %s", arg.Name)
		}

		switch arg.Type {
		case types.ToolInputTypeString:
			// String type accepts any value
		case types.ToolInputTypeNumeric:
			if _, err := strconv.ParseFloat(val, 64); err != nil {
				return fmt.Errorf("input argument '%s' must be numeric", arg.Name)
			}
		case types.ToolInputTypeBoolean:
			if _, err := strconv.ParseBool(val); err != nil {
				return fmt.Errorf("input argument '%s' must be boolean", arg.Name)
			}
		default:
			return fmt.Errorf("unknown input argument type for '%s'", arg.Name)
		}
	}
	return nil
}
