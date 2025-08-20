package enable

import "github.com/spf13/cobra"

var RootCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable resources",
}

func init() {
	RootCmd.AddCommand(problemCmd)
	RootCmd.AddCommand(tokenCmd)
}
