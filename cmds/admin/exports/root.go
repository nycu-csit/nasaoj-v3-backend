package exports

import "github.com/spf13/cobra"

var RootCmd = &cobra.Command{
	Use:   "export",
	Short: "Export resources",
}

func init() {
}
