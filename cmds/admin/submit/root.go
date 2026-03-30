package submit

import "github.com/spf13/cobra"

var RootCmd = &cobra.Command{
	Use:   "submit",
	Short: "Submit for problems / tools",
}

func init() {
	RootCmd.PersistentFlags().
		Int64("owner-id", 0, "Submit As OwnerId")
	RootCmd.MarkFlagRequired("owner-id")

	RootCmd.AddCommand(ProblemCmd)
	RootCmd.AddCommand(ToolCmd)
}
