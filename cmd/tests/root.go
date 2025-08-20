package tests

import "github.com/spf13/cobra"

var RootCmd = &cobra.Command{
	Use:   "test",
	Short: "Test problems and testcases",
}

func init() {
	RootCmd.AddCommand(stageCmd)
	RootCmd.AddCommand(problemCmd)
	RootCmd.AddCommand(depsCmd)

	RootCmd.PersistentFlags().
		String("problems-dir", "./problems", "Directory to read problems")
	RootCmd.PersistentFlags().
		String("testcases-dir", "./testcases", "Directory to read testcases")
	RootCmd.PersistentFlags().
		String("tools-dir", "./tools", "Directory to read tools")
}
