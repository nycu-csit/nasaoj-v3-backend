package submit

import "github.com/spf13/cobra"

var RootCmd = &cobra.Command{
	Use:   "submit",
	Short: "Submit for problems / tools",
}

func init() {
	RootCmd.PersistentFlags().
		String("owner-id", "", "Submit As OwnerId")
	RootCmd.MarkFlagRequired("owner-id")

	RootCmd.AddCommand(problemCmd)
	RootCmd.AddCommand(toolCmd)
}
