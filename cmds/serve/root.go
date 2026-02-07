package serve

import "github.com/spf13/cobra"

var RootCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve service",
}

func init() {
	RootCmd.PersistentFlags().
		Bool("zap-production", true, "Use production logger")

	RootCmd.AddCommand(backendCmd)
	RootCmd.AddCommand(workerCmd)
}
