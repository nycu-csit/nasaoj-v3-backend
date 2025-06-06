package serve

import "github.com/spf13/cobra"

var RootCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve service",
}

func init() {
	RootCmd.AddCommand(backendCmd)
	RootCmd.AddCommand(workerCmd)
}
