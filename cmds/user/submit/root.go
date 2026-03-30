package submit

import (
	"github.com/spf13/cobra"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/cmds/user"
)

var RootCmd = &cobra.Command{
	Use:   "submit",
	Short: "Submit problems / tools",
}

func init() {
	RootCmd.AddCommand(problemCmd)
	RootCmd.AddCommand(toolCmd)

	user.RootCmd.AddCommand(RootCmd)
}
