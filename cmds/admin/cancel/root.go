package cancel

import "github.com/spf13/cobra"

var RootCmd = &cobra.Command{
	Use:   "cancel",
	Short: "Cancel resources",
}

func init() {
	RootCmd.AddCommand(submissionCmd)
}
