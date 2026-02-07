package gen

import "github.com/spf13/cobra"

var RootCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generate resources",
}

func init() {
	RootCmd.AddCommand(tokenCmd)
}
