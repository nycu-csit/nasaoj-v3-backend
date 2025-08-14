package gen

import "github.com/spf13/cobra"

var RootCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generate Resources",
}

func init() {
	RootCmd.AddCommand(tokenCmd)
}
