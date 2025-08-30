package submit

import "github.com/spf13/cobra"

var RootCmd = &cobra.Command{
	Use:   "submit",
	Short: "Submit for problems / tools",
}

func init() {
	RootCmd.PersistentFlags().
		String("csid", "", "Submit As CsId")
	RootCmd.MarkFlagRequired("csid")

	RootCmd.AddCommand(problemCmd)
	RootCmd.AddCommand(toolCmd)
}
